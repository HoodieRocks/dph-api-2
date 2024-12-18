package routes

import (
	"context"
	"github.com/HoodieRocks/dph-api-2/auth"
	"github.com/HoodieRocks/dph-api-2/utils/paging"
	"net/http"
	"time"

	"github.com/HoodieRocks/dph-api-2/utils"
	"github.com/HoodieRocks/dph-api-2/utils/db"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

func listPendingReview(c echo.Context) error {
	// Parse the query parameters for pagination
	limit, offset, paginationErr := paging.GetPaginationModel(c)
	if paginationErr != nil {
		return paginationErr
	}

	// Establish a connection to the database
	conn := db.EstablishConnection()
	projects, err := conn.GetProjectByStatus(StatusPending, limit, offset)

	if err != nil {
		log.Errorf("Query failed: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch projects")
	}

	return c.JSON(http.StatusOK, projects)
}

func changeProjectStatus(c echo.Context) error {
	// Get the project ID and status from the request parameters
	id := c.Param("id")
	status := c.FormValue("status")

	if status == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing status")
	}

	// Establish a connection to the database
	conn := db.EstablishConnection()

	// Get the project with the given ID
	project, err := conn.GetProjectByID(id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no project found")
		}
	}

	// Start a transaction
	tx, err := conn.Db.Begin(context.Background())
	if err != nil {
		newErr := tx.Rollback(context.Background())

		if newErr != nil {
			log.Errorf("failed to rollback transaction: %v\n", newErr)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
		}
		log.Errorf("failed to create project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	// Update the project status
	err = conn.UpdateProjectStatus(tx, project.ID, status)
	if err != nil {
		newErr := tx.Rollback(context.Background())

		if newErr != nil {
			log.Errorf("failed to rollback transaction: %v\n", newErr)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
		}
		log.Errorf("failed to create project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	// Commit the transaction
	err = tx.Commit(context.Background())

	// If the commit failed, rollback and return a 500 error.
	if err != nil {
		newErr := tx.Rollback(context.Background())

		if newErr != nil {
			log.Errorf("failed to rollback transaction: %v\n", newErr)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
		}
		log.Errorf("failed to create project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	// Return a success message
	return c.String(http.StatusOK, "status updated")
}

func featureProject(c echo.Context) error {
	// Get the project ID from the request parameters
	id := c.Param("id")
	durationStr := c.QueryParam("duration")

	if durationStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing duration")
	}

	duration, err := time.ParseDuration(durationStr)

	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid duration")
	}

	// Establish a connection to the database
	conn := db.EstablishConnection()

	// Get the project with the given ID
	project, err := conn.GetProjectByID(id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no project found")
		}
	}

	// Start a transaction
	tx, err := conn.Db.Begin(context.Background())
	if err != nil {
		log.Errorf("failed to feature project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to feature project")
	}

	// Update the project status
	err = conn.FeatureProject(tx, project.ID, duration)
	if err != nil {

		newErr := tx.Rollback(context.Background())

		if newErr != nil {
			log.Errorf("failed to rollback transaction: %v\n", newErr)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to feature project")
		}

		log.Errorf("failed to feature project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to feature project")
	}

	// Commit the transaction
	err = tx.Commit(context.Background())

	// If the commit failed, rollback and return a 500 error.
	if err != nil {
		log.Errorf("failed to feature project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to feature project")
	}

	// Return a success message
	return c.String(http.StatusOK, "project featured")
}

func RegisterAdminRoutes(e *echo.Echo) {
	// main entrypoint guard
	e.Pre(auth.AllowRoles(auth.AdminRole, auth.ModeratorRole))

	e.GET("/admin/pending", listPendingReview, utils.DevRateLimiter(10))
	e.PUT("/admin/projects/:id/status", changeProjectStatus, utils.DevRateLimiter(10))
	e.POST("/admin/projects/:id/feature", featureProject, utils.DevRateLimiter(1))
}
