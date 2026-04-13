package notification

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Handler holds HTTP handler methods for notification rules.
type Handler struct {
	repo Repository
}

// NewHandler constructs a notification Handler.
func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// ListRules handles GET /api/v1/projects/:pid/jobs/:id/notifications.
func (h *Handler) ListRules(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	rules, err := h.repo.ListByJob(c.Request().Context(), jobID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list notification rules")
	}
	if rules == nil {
		rules = []*NotificationRule{}
	}
	return c.JSON(http.StatusOK, rules)
}

type createRuleRequest struct {
	Event   NotificationEvent      `json:"event"`
	Channel NotificationChannel    `json:"channel"`
	Config  map[string]interface{} `json:"config"`
}

// CreateRule handles POST /api/v1/projects/:pid/jobs/:id/notifications.
func (h *Handler) CreateRule(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	var req createRuleRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Event == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "event is required")
	}
	if req.Channel == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "channel is required")
	}

	rule := &NotificationRule{
		ID:      uuid.New(),
		JobID:   jobID,
		Event:   req.Event,
		Channel: req.Channel,
		Config:  req.Config,
	}

	if err := h.repo.Create(c.Request().Context(), rule); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create notification rule")
	}

	return c.JSON(http.StatusCreated, rule)
}

type updateRuleRequest struct {
	Event   NotificationEvent      `json:"event"`
	Channel NotificationChannel    `json:"channel"`
	Config  map[string]interface{} `json:"config"`
}

// UpdateRule handles PUT /api/v1/projects/:pid/jobs/:id/notifications/:rid.
func (h *Handler) UpdateRule(c echo.Context) error {
	rid, err := uuid.Parse(c.Param("rid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid rule id")
	}

	ctx := c.Request().Context()
	rule, err := h.repo.FindByID(ctx, rid)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch rule")
	}
	if rule == nil {
		return echo.NewHTTPError(http.StatusNotFound, "notification rule not found")
	}

	var req updateRuleRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Event != "" {
		rule.Event = req.Event
	}
	if req.Channel != "" {
		rule.Channel = req.Channel
	}
	if req.Config != nil {
		rule.Config = req.Config
	}

	if err := h.repo.Update(ctx, rule); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update notification rule")
	}

	return c.JSON(http.StatusOK, rule)
}

// DeleteRule handles DELETE /api/v1/projects/:pid/jobs/:id/notifications/:rid.
func (h *Handler) DeleteRule(c echo.Context) error {
	rid, err := uuid.Parse(c.Param("rid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid rule id")
	}

	if err := h.repo.Delete(c.Request().Context(), rid); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete notification rule")
	}

	return c.NoContent(http.StatusNoContent)
}
