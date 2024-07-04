package utils

import (
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
