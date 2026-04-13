package schedule

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Handler holds HTTP handler methods for job schedules.
type Handler struct {
	repo      Repository
	scheduler *Scheduler
}

// NewHandler constructs a schedule Handler.
func NewHandler(repo Repository, scheduler *Scheduler) *Handler {
	return &Handler{repo: repo, scheduler: scheduler}
}

// GetSchedule handles GET /api/v1/projects/:pid/jobs/:id/schedule.
func (h *Handler) GetSchedule(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	sc, err := h.repo.FindByJobID(c.Request().Context(), jobID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch schedule")
	}
	if sc == nil {
		return echo.NewHTTPError(http.StatusNotFound, "schedule not found")
	}

	return c.JSON(http.StatusOK, sc)
}

type upsertScheduleRequest struct {
	CronExpr  string `json:"cron_expr"`
	Timezone  string `json:"timezone"`
	IsEnabled *bool  `json:"is_enabled"`
}

// UpsertSchedule handles PUT /api/v1/projects/:pid/jobs/:id/schedule.
func (h *Handler) UpsertSchedule(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	var req upsertScheduleRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.CronExpr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "cron_expr is required")
	}

	// Validate cron expression.
	if _, err := parseCronExpr(req.CronExpr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid cron_expr: "+err.Error())
	}

	tz := req.Timezone
	if tz == "" {
		tz = "UTC"
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	ctx := c.Request().Context()

	// Fetch existing or build new.
	sc, err := h.repo.FindByJobID(ctx, jobID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch schedule")
	}
	if sc == nil {
		sc = &Schedule{ID: uuid.New(), JobID: jobID}
	}

	sc.CronExpr = req.CronExpr
	sc.Timezone = tz
	sc.IsEnabled = isEnabled

	if err := h.repo.Upsert(ctx, sc); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save schedule")
	}

	// Update in-memory scheduler.
	if isEnabled {
		if err := h.scheduler.RegisterJob(ctx, sc); err != nil {
			// Log but don't fail - schedule is saved in DB.
			c.Logger().Errorf("register schedule in cron: %v", err)
		}
	} else {
		h.scheduler.UnregisterJob(sc.ID.String())
	}

	return c.JSON(http.StatusOK, sc)
}

// DeleteSchedule handles DELETE /api/v1/projects/:pid/jobs/:id/schedule.
func (h *Handler) DeleteSchedule(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	ctx := c.Request().Context()
	sc, err := h.repo.FindByJobID(ctx, jobID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch schedule")
	}
	if sc == nil {
		return echo.NewHTTPError(http.StatusNotFound, "schedule not found")
	}

	if err := h.repo.Delete(ctx, sc.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete schedule")
	}

	h.scheduler.UnregisterJob(sc.ID.String())

	return c.NoContent(http.StatusNoContent)
}
