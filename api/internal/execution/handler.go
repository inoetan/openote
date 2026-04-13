package execution

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"github.com/inoetan/openote/api/internal/auth"
	"github.com/inoetan/openote/api/pkg/models"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handler holds HTTP handler methods for executions.
type Handler struct {
	repo  Repository
	redis *redis.Client
}

// NewHandler constructs an execution Handler.
func NewHandler(repo Repository, redisClient *redis.Client) *Handler {
	return &Handler{repo: repo, redis: redisClient}
}

// Execute handles POST /api/v1/projects/:pid/jobs/:id/execute.
func (h *Handler) Execute(c echo.Context) error {
	pid, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	claims := auth.ClaimsFromContext(c)
	var triggeredUser *uuid.UUID
	if claims != nil {
		uid, err := uuid.Parse(claims.UserID)
		if err == nil {
			triggeredUser = &uid
		}
	}

	ctx := c.Request().Context()
	e := &Execution{
		ID:            uuid.New(),
		JobID:         jobID,
		ProjectID:     pid,
		TriggeredBy:   TriggerManual,
		TriggeredUser: triggeredUser,
		Status:        StatusQueued,
		CreatedAt:     time.Now().UTC(),
	}

	if err := h.repo.Create(ctx, e); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create execution")
	}

	dispatch := models.JobDispatch{
		ExecutionID: e.ID.String(),
		JobID:       jobID.String(),
		ProjectID:   pid.String(),
	}
	payload, err := json.Marshal(dispatch)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to marshal dispatch")
	}

	if err := h.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: models.RedisStreamKey,
		Values: map[string]interface{}{
			"payload": string(payload),
		},
	}).Err(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to dispatch execution")
	}

	return c.JSON(http.StatusCreated, e)
}

// ListExecutions handles GET /api/v1/projects/:pid/executions.
func (h *Handler) ListExecutions(c echo.Context) error {
	pid, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}

	limit := 20
	offset := 0
	if l := c.QueryParam("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	executions, err := h.repo.ListByProject(c.Request().Context(), pid, limit, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list executions")
	}
	if executions == nil {
		executions = []*Execution{}
	}
	return c.JSON(http.StatusOK, executions)
}

// GetExecution handles GET /api/v1/projects/:pid/executions/:eid.
func (h *Handler) GetExecution(c echo.Context) error {
	eid, err := uuid.Parse(c.Param("eid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid execution id")
	}

	e, err := h.repo.FindByID(c.Request().Context(), eid)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch execution")
	}
	if e == nil {
		return echo.NewHTTPError(http.StatusNotFound, "execution not found")
	}

	return c.JSON(http.StatusOK, e)
}

// AbortExecution handles POST /api/v1/projects/:pid/executions/:eid/abort.
func (h *Handler) AbortExecution(c echo.Context) error {
	eid, err := uuid.Parse(c.Param("eid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid execution id")
	}

	ctx := c.Request().Context()
	e, err := h.repo.FindByID(ctx, eid)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch execution")
	}
	if e == nil {
		return echo.NewHTTPError(http.StatusNotFound, "execution not found")
	}

	if e.Status != StatusQueued && e.Status != StatusRunning {
		return echo.NewHTTPError(http.StatusConflict, "execution is not in an abortable state")
	}

	now := time.Now().UTC()
	if err := h.repo.UpdateStatus(ctx, eid, StatusAborted, &now, nil); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to abort execution")
	}

	// Notify workers via Redis PubSub.
	channel := models.RedisPubSubPrefix + eid.String()
	_ = h.redis.Publish(ctx, channel, `{"action":"abort"}`).Err()

	e.Status = StatusAborted
	e.FinishedAt = &now
	return c.JSON(http.StatusOK, e)
}

// StreamLogs handles GET /api/v1/executions/:eid/logs/stream (WebSocket).
// It subscribes to the Redis PubSub channel for real-time log lines.
func (h *Handler) StreamLogs(c echo.Context) error {
	eid, err := uuid.Parse(c.Param("eid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid execution id")
	}

	conn, err := wsUpgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return nil // Upgrade itself wrote the error response.
	}
	defer conn.Close()

	ctx := c.Request().Context()
	channel := models.RedisPubSubPrefix + eid.String()
	pubsub := h.redis.Subscribe(ctx, channel)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				return nil
			}
		}
	}
}
