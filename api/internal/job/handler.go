package job

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/inoetan/openote/api/internal/auth"
)

// Handler holds HTTP handler methods for node definitions and jobs.
type Handler struct {
	nodeRepo NodeRepository
	jobRepo  JobRepository
}

// NewHandler constructs a job Handler.
func NewHandler(nodeRepo NodeRepository, jobRepo JobRepository) *Handler {
	return &Handler{nodeRepo: nodeRepo, jobRepo: jobRepo}
}

// ---- Node Definition handlers ----

// ListNodes handles GET /api/v1/projects/:pid/nodes.
func (h *Handler) ListNodes(c echo.Context) error {
	pid, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}

	nodes, err := h.nodeRepo.ListByProject(c.Request().Context(), pid)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list nodes")
	}
	if nodes == nil {
		nodes = []*NodeDefinition{}
	}
	return c.JSON(http.StatusOK, nodes)
}

type createNodeRequest struct {
	Name        string                 `json:"name"`
	Type        NodeType               `json:"type"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
}

// CreateNode handles POST /api/v1/projects/:pid/nodes.
func (h *Handler) CreateNode(c echo.Context) error {
	pid, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}

	claims := auth.ClaimsFromContext(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid user id in token")
	}

	var req createNodeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}
	if req.Type == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "type is required")
	}

	nd := &NodeDefinition{
		ID:          uuid.New(),
		ProjectID:   pid,
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		Config:      req.Config,
		CreatedBy:   userID,
	}

	if err := h.nodeRepo.Create(c.Request().Context(), nd); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create node")
	}

	return c.JSON(http.StatusCreated, nd)
}

// GetNode handles GET /api/v1/projects/:pid/nodes/:id.
func (h *Handler) GetNode(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid node id")
	}

	nd, err := h.nodeRepo.FindByID(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch node")
	}
	if nd == nil {
		return echo.NewHTTPError(http.StatusNotFound, "node not found")
	}
	return c.JSON(http.StatusOK, nd)
}

type updateNodeRequest struct {
	Name        string                 `json:"name"`
	Type        NodeType               `json:"type"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
}

// UpdateNode handles PUT /api/v1/projects/:pid/nodes/:id.
func (h *Handler) UpdateNode(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid node id")
	}

	ctx := c.Request().Context()
	nd, err := h.nodeRepo.FindByID(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch node")
	}
	if nd == nil {
		return echo.NewHTTPError(http.StatusNotFound, "node not found")
	}

	var req updateNodeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Name != "" {
		nd.Name = req.Name
	}
	if req.Type != "" {
		nd.Type = req.Type
	}
	nd.Description = req.Description
	if req.Config != nil {
		nd.Config = req.Config
	}

	if err := h.nodeRepo.Update(ctx, nd); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update node")
	}

	return c.JSON(http.StatusOK, nd)
}

// DeleteNode handles DELETE /api/v1/projects/:pid/nodes/:id.
func (h *Handler) DeleteNode(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid node id")
	}

	if err := h.nodeRepo.Delete(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete node")
	}

	return c.NoContent(http.StatusNoContent)
}

// ---- Job handlers ----

// ListJobs handles GET /api/v1/projects/:pid/jobs.
func (h *Handler) ListJobs(c echo.Context) error {
	pid, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}

	jobs, err := h.jobRepo.ListByProject(c.Request().Context(), pid)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list jobs")
	}
	if jobs == nil {
		jobs = []*Job{}
	}
	return c.JSON(http.StatusOK, jobs)
}

type createJobRequest struct {
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	IsActive      *bool        `json:"is_active"`
	TimeoutSecs   int          `json:"timeout_secs"`
	MaxConcurrent int          `json:"max_concurrent"`
	OnFailure     OnFailure    `json:"on_failure"`
	RetryCount    int          `json:"retry_count"`
	WorkflowSpec  WorkflowSpec `json:"workflow_spec"`
}

// CreateJob handles POST /api/v1/projects/:pid/jobs.
func (h *Handler) CreateJob(c echo.Context) error {
	pid, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}

	claims := auth.ClaimsFromContext(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid user id in token")
	}

	var req createJobRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	onFailure := req.OnFailure
	if onFailure == "" {
		onFailure = OnFailureStop
	}

	j := &Job{
		ID:            uuid.New(),
		ProjectID:     pid,
		Name:          req.Name,
		Description:   req.Description,
		IsActive:      isActive,
		TimeoutSecs:   req.TimeoutSecs,
		MaxConcurrent: req.MaxConcurrent,
		OnFailure:     onFailure,
		RetryCount:    req.RetryCount,
		WorkflowSpec:  req.WorkflowSpec,
		CreatedBy:     userID,
	}

	if err := h.jobRepo.Create(c.Request().Context(), j); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create job")
	}

	return c.JSON(http.StatusCreated, j)
}

// GetJob handles GET /api/v1/projects/:pid/jobs/:id.
func (h *Handler) GetJob(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	j, err := h.jobRepo.FindByID(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch job")
	}
	if j == nil {
		return echo.NewHTTPError(http.StatusNotFound, "job not found")
	}
	return c.JSON(http.StatusOK, j)
}

type updateJobRequest struct {
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	IsActive      *bool        `json:"is_active"`
	TimeoutSecs   int          `json:"timeout_secs"`
	MaxConcurrent int          `json:"max_concurrent"`
	OnFailure     OnFailure    `json:"on_failure"`
	RetryCount    int          `json:"retry_count"`
	WorkflowSpec  WorkflowSpec `json:"workflow_spec"`
}

// UpdateJob handles PUT /api/v1/projects/:pid/jobs/:id.
func (h *Handler) UpdateJob(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	ctx := c.Request().Context()
	j, err := h.jobRepo.FindByID(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch job")
	}
	if j == nil {
		return echo.NewHTTPError(http.StatusNotFound, "job not found")
	}

	var req updateJobRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Name != "" {
		j.Name = req.Name
	}
	j.Description = req.Description
	if req.IsActive != nil {
		j.IsActive = *req.IsActive
	}
	j.TimeoutSecs = req.TimeoutSecs
	j.MaxConcurrent = req.MaxConcurrent
	if req.OnFailure != "" {
		j.OnFailure = req.OnFailure
	}
	j.RetryCount = req.RetryCount
	j.WorkflowSpec = req.WorkflowSpec

	if err := h.jobRepo.Update(ctx, j); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update job")
	}

	return c.JSON(http.StatusOK, j)
}

// DeleteJob handles DELETE /api/v1/projects/:pid/jobs/:id.
func (h *Handler) DeleteJob(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	if err := h.jobRepo.Delete(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete job")
	}

	return c.NoContent(http.StatusNoContent)
}
