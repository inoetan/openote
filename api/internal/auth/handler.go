package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/inoetan/openote/api/internal/user"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
	refreshKeyPrefix = "refresh:"
)

// Handler holds dependencies for auth HTTP handlers.
type Handler struct {
	users     user.Repository
	redis     *redis.Client
	jwtSecret string
}

// NewHandler constructs an auth Handler.
func NewHandler(users user.Repository, redisClient *redis.Client, jwtSecret string) *Handler {
	return &Handler{
		users:     users,
		redis:     redisClient,
		jwtSecret: jwtSecret,
	}
}

type loginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Username == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "username and password are required")
	}

	u, err := h.users.FindByUsername(c.Request().Context(), req.Username)
	if err != nil || u == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}
	if !u.IsActive {
		return echo.NewHTTPError(http.StatusUnauthorized, "account is disabled")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	accessToken, err := GenerateToken(u.ID.String(), u.Username, nil, h.jwtSecret, accessTokenTTL)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "could not generate token")
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "could not generate refresh token")
	}

	// Store refresh token in Redis: refresh:<token> → userID
	ctx := c.Request().Context()
	if err := h.redis.Set(ctx, refreshKeyPrefix+refreshToken, u.ID.String(), refreshTokenTTL).Err(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "could not store refresh token")
	}

	return c.JSON(http.StatusOK, tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(accessTokenTTL.Seconds()),
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Refresh handles POST /api/v1/auth/refresh.
func (h *Handler) Refresh(c echo.Context) error {
	var req refreshRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.RefreshToken == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "refresh_token is required")
	}

	ctx := c.Request().Context()
	key := refreshKeyPrefix + req.RefreshToken

	userID, err := h.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired refresh token")
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "token lookup failed")
	}

	// Delete old refresh token (rotation)
	_ = h.redis.Del(ctx, key)

	u, err := h.users.FindByID(ctx, userID)
	if err != nil || u == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not found")
	}
	if !u.IsActive {
		return echo.NewHTTPError(http.StatusUnauthorized, "account is disabled")
	}

	accessToken, err := GenerateToken(u.ID.String(), u.Username, nil, h.jwtSecret, accessTokenTTL)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "could not generate token")
	}

	newRefresh, err := generateRefreshToken()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "could not generate refresh token")
	}

	if err := h.redis.Set(ctx, refreshKeyPrefix+newRefresh, u.ID.String(), refreshTokenTTL).Err(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "could not store refresh token")
	}

	return c.JSON(http.StatusOK, tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
		ExpiresIn:    int(accessTokenTTL.Seconds()),
	})
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Logout handles POST /api/v1/auth/logout.
func (h *Handler) Logout(c echo.Context) error {
	var req logoutRequest
	_ = c.Bind(&req) // best-effort

	if req.RefreshToken != "" {
		ctx := context.Background()
		_ = h.redis.Del(ctx, refreshKeyPrefix+req.RefreshToken)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "logged out"})
}

// generateRefreshToken produces a cryptographically secure 32-byte hex token.
func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
