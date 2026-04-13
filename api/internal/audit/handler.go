package audit

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Handler holds HTTP handler methods for audit logs.
type Handler struct {
	repo Repository
}

// NewHandler constructs an audit Handler.
func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// ListAuditEvents handles GET /api/v1/projects/:pid/audit.
func (h *Handler) ListAuditEvents(c echo.Context) error {
	pid, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}

	limit := 50
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

	events, err := h.repo.ListByProject(c.Request().Context(), pid, limit, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list audit events")
	}
	if events == nil {
		events = []*AuditEvent{}
	}
	return c.JSON(http.StatusOK, events)
}
