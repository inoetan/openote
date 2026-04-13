package auth

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

const (
	// ContextKeyUserID is the echo.Context key for the authenticated user's ID.
	ContextKeyUserID = "user_id"
	// ContextKeyUsername is the echo.Context key for the authenticated username.
	ContextKeyUsername = "username"
	// ContextKeyClaims is the echo.Context key for the full *Claims object.
	ContextKeyClaims = "claims"
)

// JWTMiddleware returns an Echo middleware that validates Bearer tokens.
// On success it stores user_id, username, and claims in the echo.Context.
func JWTMiddleware(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header format")
			}

			claims, err := ValidateToken(parts[1], secret)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyUsername, claims.Username)
			c.Set(ContextKeyClaims, claims)

			return next(c)
		}
	}
}

// ClaimsFromContext extracts the *Claims stored by JWTMiddleware.
func ClaimsFromContext(c echo.Context) *Claims {
	v := c.Get(ContextKeyClaims)
	if v == nil {
		return nil
	}
	claims, _ := v.(*Claims)
	return claims
}
