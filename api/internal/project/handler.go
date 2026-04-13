package project

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/inoetan/openote/api/internal/auth"
)

// Handler holds HTTP handler methods for project management.
type Handler struct {
	repo Repository
}

// NewHandler constructs a project Handler.
func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// projectResponse is the API representation of a project.
type projectResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toProjectResponse(p *Project) projectResponse {
	return projectResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		CreatedBy:   p.CreatedBy,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// List handles GET /api/v1/projects.
func (h *Handler) List(c echo.Context) error {
	claims := auth.ClaimsFromContext(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid user id in token")
	}

	projects, err := h.repo.List(c.Request().Context(), userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list projects")
	}

	resp := make([]projectResponse, 0, len(projects))
	for _, p := range projects {
		resp = append(resp, toProjectResponse(p))
	}
	return c.JSON(http.StatusOK, resp)
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Create handles POST /api/v1/projects.
func (h *Handler) Create(c echo.Context) error {
	claims := auth.ClaimsFromContext(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid user id in token")
	}

	var req createProjectRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	p := &Project{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   userID,
	}

	if err := h.repo.Create(c.Request().Context(), p); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	return c.JSON(http.StatusCreated, toProjectResponse(p))
}

type updateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Get handles GET /api/v1/projects/:id.
func (h *Handler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}

	p, err := h.repo.FindByID(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project")
	}
	if p == nil {
		return echo.NewHTTPError(http.StatusNotFound, "project not found")
	}

	return c.JSON(http.StatusOK, toProjectResponse(p))
}

// Update handles PUT /api/v1/projects/:id.
func (h *Handler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}

	ctx := c.Request().Context()
	p, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project")
	}
	if p == nil {
		return echo.NewHTTPError(http.StatusNotFound, "project not found")
	}

	var req updateProjectRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Name != "" {
		p.Name = req.Name
	}
	p.Description = req.Description

	if err := h.repo.Update(ctx, p); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update project")
	}

	return c.JSON(http.StatusOK, toProjectResponse(p))
}

// Delete handles DELETE /api/v1/projects/:id.
func (h *Handler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}

	if err := h.repo.Delete(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete project")
	}

	return c.NoContent(http.StatusNoContent)
}
