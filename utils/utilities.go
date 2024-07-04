package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"me/cobble/utils/db"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

func DevRateLimiter(rps rate.Limit) echo.MiddlewareFunc {

	if os.Getenv("BENCHMARK") == "true" {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				return next(c)
			}
		}
	}

	return middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(rps))
}

func IsUserProjectOwner(project db.Project, token *string, validToken bool) (bool, error) {

	conn := db.EstablishConnection()

	// If the project is draft, check if the user is the owner and has a valid token.
	owner, err := conn.GetUserById(project.Author)

	if !validToken || token == nil {
		// If the user does not have a valid token, return a forbidden error.
		return false, echo.NewHTTPError(http.StatusForbidden, "invalid or expired token")
	}

	if err != nil {
		if err == pgx.ErrNoRows {
			return false, echo.NewHTTPError(http.StatusNotFound, "no owner found")
		}
		fmt.Fprintf(os.Stderr, "failed to fetch project owner: %v\n", err)
		return false, echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project owner")
	}

	user, err := conn.GetUserByToken(*token)

	if err != nil {
		if err == pgx.ErrNoRows {
			return false, echo.NewHTTPError(http.StatusNotFound, "no user is assigned to this token")
		}
		fmt.Fprintf(os.Stderr, "failed to fetch project owner: %v\n", err)
		return false, echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project owner")
	}

	return owner.ID == user.ID, nil
}

func ValidateToken(token string) (bool, *string) {
	var tokenParts = strings.Split(token, " ")

	if len(tokenParts) < 2 {
		return false, nil
	}

	return tokenParts[0] == "Bearer" && len(tokenParts[1]) > 8, &tokenParts[1]
}

func GenerateSecureToken() string {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
