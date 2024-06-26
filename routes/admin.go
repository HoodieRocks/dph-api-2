package routes

import (
	"context"
	"fmt"
	utils "me/cobble/utils/db"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

func adminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Establish a connection to the database
		conn := utils.EstablishConnection()

		// Get the token from the authorization header
		rawToken := c.Request().Header.Get(echo.HeaderAuthorization)

		// Validate the token
		validToken, token := utils.TokenValidate(rawToken)
		if !validToken || token == nil {
			return echo.NewHTTPError(http.StatusBadRequest, "malformed token")
		}		
		
		mod, err := conn.GetUserByToken(*token)
		if err != nil {
			if err == pgx.ErrNoRows {
				return echo.NewHTTPError(http.StatusForbidden, "invalid token")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch privileged user information")
		}
		c.Set("mod", mod)
		return next(c)
	}
}


func listPendingReview(c echo.Context) error {
	var mod utils.User = c.Get("mod").(utils.User)

	// Check if the user has permission to change the project status
	if mod.Role != "moderator" && mod.Role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "you do not have permission to view this")
	}

	// Establish a connection to the database
	conn := utils.EstablishConnection()

	rows, err := conn.Db.Query(context.Background(), "SELECT * FROM projects WHERE status = 'pending'")

	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch projects")
	}

	projects, err := pgx.CollectRows(rows, pgx.RowToStructByName[utils.Project])

	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch projects")
	}

	return c.JSON(http.StatusOK, projects)
}

func RegisterAdminRoutes(e *echo.Echo) {
	e.GET("/admin", listPendingReview, adminMiddleware)
}