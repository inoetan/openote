package user

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// Handler holds HTTP handler methods for user management.
type Handler struct {
	repo Repository
}

// NewHandler constructs a user Handler.
func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// userResponse is the API representation of a user (no password hash).
type userResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toUserResponse(u *User) userResponse {
	return userResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// List handles GET /api/v1/users.
func (h *Handler) List(c echo.Context) error {
	users, err := h.repo.List(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list users")
	}

	resp := make([]userResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, toUserResponse(u))
	}
	return c.JSON(http.StatusOK, resp)
}

type createUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	IsActive *bool  `json:"is_active"`
}

// Create handles POST /api/v1/users.
func (h *Handler) Create(c echo.Context) error {
	var req createUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "username, email, and password are required")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to hash password")
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	u := &User{
		ID:           uuid.New(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		IsActive:     isActive,
	}

	if err := h.repo.Create(c.Request().Context(), u); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
	}

	return c.JSON(http.StatusCreated, toUserResponse(u))
}

type updateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	IsActive *bool  `json:"is_active"`
}

// Update handles PUT /api/v1/users/:id.
func (h *Handler) Update(c echo.Context) error {
	idStr := c.Param("id")
	uid, err := uuid.Parse(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	ctx := c.Request().Context()
	u, err := h.repo.FindByID(ctx, uid.String())
	if err != nil || u == nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	var req updateUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Email != "" {
		u.Email = req.Email
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to hash password")
		}
		u.PasswordHash = string(hash)
	}
	if req.IsActive != nil {
		u.IsActive = *req.IsActive
	}

	if err := h.repo.Update(ctx, u); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update user")
	}

	return c.JSON(http.StatusOK, toUserResponse(u))
}

// Delete handles DELETE /api/v1/users/:id.
func (h *Handler) Delete(c echo.Context) error {
	idStr := c.Param("id")
	uid, err := uuid.Parse(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	if err := h.repo.Delete(c.Request().Context(), uid); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete user")
	}

	return c.NoContent(http.StatusNoContent)
}
