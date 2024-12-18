package utils

import (
	"crypto/rand"
	"encoding/base64"
	"os"

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

func GenerateSecureToken() string {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		//TODO empty token potentially written to user profile...
		//TODO security hotspot GenerateSecureToken
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
