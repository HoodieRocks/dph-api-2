package routes

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/HoodieRocks/dph-api-2/utils"
	"github.com/HoodieRocks/dph-api-2/utils/db"
	"github.com/HoodieRocks/dph-api-2/utils/files"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

func listProjects(c echo.Context) error {
	// Define the structure of the search results.
	type SearchResults struct {
		Time    float64      `json:"time"` // Search time in seconds
		Count   int          `json:"count"`
		Results []db.Project `json:"results"`
	}

	// Establish a connection to the database
	var conn = db.EstablishConnection()

	// Parse the query parameters for pagination
	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil {
		page = 0
	}

	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil {
		limit = 25
	}

	offset := page * limit

	var startTime = time.Now()
	// Retrieve the projects from the database
	results, err := conn.ListProjects(limit, offset, "downloads")
	if err != nil {
		log.Errorf("failed to fetch projects: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch projects")
	}

	// Return the projects as JSON
	return c.JSON(http.StatusOK, SearchResults{
		Time:    time.Since(startTime).Seconds(),
		Count:   len(results),
		Results: results,
	})
}

func getProjectById(c echo.Context) error {
	// Get the project ID from the URL parameter.
	id := c.Param("id")

	// Establish a connection to the database.
	var conn = db.EstablishConnection()

	// Get the project from the database.
	project, err := conn.GetProjectByID(id)

	// If there was an error fetching the project, return an appropriate error.
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no project found")
		}
		log.Errorf("failed to fetch project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project")
	}

	// Get the token from the request headers.
	rawToken := c.Request().Header.Get(echo.HeaderAuthorization)
	validToken, token := utils.ValidateToken(rawToken)

	// Check the status of the project.
	switch project.Status {
	case "live":
		// If the project is live, return the project.
		return c.JSON(http.StatusOK, project)
	case "draft":
		isOwner, err := utils.IsUserProjectOwner(project, token, validToken)

		if err != nil {
			return err
		}

		if isOwner {
			// If the user is the owner, return the project.
			return c.JSON(http.StatusOK, project)
		} else {
			// If the user is not the owner, return a forbidden error.
			return echo.NewHTTPError(http.StatusForbidden, "you can not access other's private projects")
		}
	case "pending":
		isOwner, err := utils.IsUserProjectOwner(project, token, validToken)

		if err != nil {
			return err
		}

		conn := db.EstablishConnection()

		user, err := conn.GetUserByToken(*token)

		if err != nil {
			if err == pgx.ErrNoRows {
				return echo.NewHTTPError(http.StatusNotFound, "no user is assigned to this token")
			}
			log.Errorf("failed to fetch project owner: %v\n", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project owner")
		}

		if isOwner || (user.Role == "admin" || user.Role == "moderator") {
			// If the user is the owner, return the project.
			return c.JSON(http.StatusOK, project)
		} else {
			// If the user is not the owner, return a forbidden error.
			return echo.NewHTTPError(http.StatusForbidden, "you can not access other's private projects")
		}
	default:
		// If the project is in an illegal state, return an internal server error.
		return echo.NewHTTPError(http.StatusInternalServerError, "illegal project state")
	}
}

func randomProject(c echo.Context) error {
	var limit, err = strconv.Atoi(c.QueryParam("limit"))

	if err != nil {
		limit = 1
	}

	var conn = db.EstablishConnection()

	project, err := conn.GetRandomProjects(limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project")
	}
	return c.JSON(http.StatusOK, project)
}

func getProjectBySlug(c echo.Context) error {
	// Get the project ID from the URL parameter.
	id := c.Param("slug")

	// Establish a connection to the database.
	var conn = db.EstablishConnection()

	// Get the project from the database.
	project, err := conn.GetProjectBySlug(id)

	// If there was an error fetching the project, return an appropriate error.
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no project found")
		}
		log.Errorf("failed to fetch project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project")
	}

	// Get the token from the request headers.
	rawToken := c.Request().Header.Get(echo.HeaderAuthorization)
	validToken, token := utils.ValidateToken(rawToken)

	// Check the status of the project.
	switch project.Status {
	case "live":
		// If the project is live, return the project.
		return c.JSON(http.StatusOK, project)
	case "draft":
		isOwner, err := utils.IsUserProjectOwner(project, token, validToken)

		if err != nil {
			return err
		}

		if isOwner {
			// If the user is the owner, return the project.
			return c.JSON(http.StatusOK, project)
		} else {
			// If the user is not the owner, return a forbidden error.
			return echo.NewHTTPError(http.StatusForbidden, "you can not access other's private projects")
		}
	case "pending":
		isOwner, err := utils.IsUserProjectOwner(project, token, validToken)

		if err != nil {
			return err
		}

		conn := db.EstablishConnection()

		user, err := conn.GetUserByToken(*token)

		if err != nil {
			if err == pgx.ErrNoRows {
				return echo.NewHTTPError(http.StatusNotFound, "no user is assigned to this token")
			}
			log.Errorf("failed to fetch project owner: %v\n", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project owner")
		}

		if isOwner || (user.Role == "admin" || user.Role == "moderator") {
			// If the user is the owner, return the project.
			return c.JSON(http.StatusOK, project)
		} else {
			// If the user is not the owner, return a forbidden error.
			return echo.NewHTTPError(http.StatusForbidden, "you can not access other's private projects")
		}
	default:
		// If the project is in an illegal state, return an internal server error.
		return echo.NewHTTPError(http.StatusInternalServerError, "illegal project state")
	}
}

func createProject(c echo.Context) error {
	// Get the token from the Authorization header
	rawToken := c.Request().Header.Get("Authorization")

	// Get the form values
	title := c.FormValue("title")
	slug := c.FormValue("slug")
	description := c.FormValue("description")
	body := c.FormValue("body")
	icon, err := c.FormFile("icon")

	// If there was an error parsing the icon, return a bad request error
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "malformed form")
	}

	// Split the category string by commas and convert it to a slice of strings
	category := strings.Split(c.FormValue("category"), ",")

	// Establish a connection to the database
	conn := db.EstablishConnection()

	// Validate the token
	validToken, token := utils.ValidateToken(rawToken)

	// Check if the token is valid
	if !validToken || token == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "malformed token")
	}

	// Check if a project with the same title or slug already exists
	projectTaken := conn.CheckForProjectNameConflict(title, slug)

	// If a project with the same title or slug already exists, return a conflict error
	if projectTaken {
		return echo.NewHTTPError(http.StatusConflict, "another project shares that title or slug")
	}

	// Get the user associated with the token
	user, err := conn.GetUserByToken(*token)

	// If the token is invalid, return a forbidden error
	if err == pgx.ErrNoRows {
		return echo.NewHTTPError(http.StatusForbidden, "invalid token")
	}

	var iconPath string

	if icon != nil {

		// Upload the icon file
		iconPath, err = files.UploadIconFile(icon, db.Project{
			Title:       title,
			Slug:        slug,
			Author:      user.ID,
			Description: description,
			Body:        body,
			Creation:    time.Now(),
			Updated:     time.Now(),
			Category:    category,
		})

		// If there was an error uploading the icon, return an internal server error
		if err != nil {
			log.Errorf("failed to upload icon: %v\n", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to upload icon")
		}
	}

	// Create the project struct
	project := db.Project{
		Title:       title,
		Slug:        slug,
		Author:      user.ID,
		Description: description,
		Body:        body,
		Creation:    time.Now(),
		Updated:     time.Now(),
		Category:    category,
		Icon:        &iconPath,
	}

	// Start a transaction
	tx, err := conn.Db.Begin(context.Background())

	// If there was an error starting the transaction, rollback and return an internal server error
	if err != nil {
		newErr := tx.Rollback(context.Background())

		if newErr != nil {
			log.Errorf("failed to rollback transaction: %v\n", newErr)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
		}

		log.Errorf("failed to start project transaction: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	// Create the project in the database
	err = conn.CreateProject(tx, project)

	// If there was an error creating the project, rollback and return an internal server error
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

	// If there was an error committing the transaction, rollback and return an internal server error
	if err != nil {
		newErr := tx.Rollback(context.Background())

		if newErr != nil {
			log.Errorf("failed to rollback transaction: %v\n", newErr)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
		}

		log.Errorf("failed to write project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	// Return the created project in JSON format
	return c.JSON(http.StatusOK, project)
}

func updateProject(c echo.Context) error {

	var pid = c.Param("id")

	// Establish a connection to the database.
	var conn = db.EstablishConnection()

	// Get the project from the database.
	project, err := conn.GetProjectBySlug(pid)

	// If there was an error fetching the project, return an appropriate error.
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no project found")
		}
		log.Errorf("failed to fetch project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project")
	}

	// Get the token from the request headers.
	rawToken := c.Request().Header.Get(echo.HeaderAuthorization)
	validToken, token := utils.ValidateToken(rawToken)

	// If the token is invalid, return a forbidden error.
	if !validToken {
		return echo.NewHTTPError(http.StatusForbidden, "invalid token")
	}

	isOwner, err := utils.IsUserProjectOwner(project, token, validToken)

	if err != nil {
		log.Errorf("failed to fetch project owner: %v\n", err)
		return echo.NewHTTPError(http.StatusForbidden, "failed to fetch project owner")
	}

	if !isOwner {
		return echo.NewHTTPError(http.StatusForbidden, "not authorized")
	}

	tx, err := conn.Db.Begin(context.Background())

	if err != nil {
		log.Errorf("failed to initialise transaction: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	if project.Status == "live" {
		err = conn.UpdateProjectStatus(tx, project.ID, "draft")

		if err != nil {
			newErr := tx.Rollback(context.Background())

			if newErr != nil {
				log.Errorf("failed to rollback transaction: %v\n", newErr)
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
			}
			log.Errorf("failed to update project: %v\n", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to update project")
		}

		err = tx.Commit(context.Background())

		if err != nil {
			newErr := tx.Rollback(context.Background())

			if newErr != nil {
				log.Errorf("failed to rollback transaction: %v\n", newErr)
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
			}
			log.Errorf("failed to commit transaction: %v\n", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
		}
	}

	return c.JSON(http.StatusOK, project)
}

func ftsSearch(c echo.Context) error {
	// Get the search query from the request parameters
	query := c.QueryParam("q")

	// Establish a connection to the database
	conn := db.EstablishConnection()

	// Record the start time of the search
	startTime := time.Now()

	// Perform the full-text search
	rows, err := conn.FTSSearchProjects(query)

	// Define the search results structure
	type SearchResults struct {
		Time    float64      `json:"time"` // Search time in seconds
		Count   int          `json:"count"`
		Results []db.Project `json:"results"`
	}

	// Handle errors during the search
	if err != nil {
		if err == pgx.ErrNoRows {
			// Return empty results if no rows are found
			return c.JSON(http.StatusOK, SearchResults{
				Time:    time.Since(startTime).Seconds(),
				Count:   0,
				Results: make([]db.Project, 0),
			})
		}

		// Log and return an error if the search fails
		log.Errorf("failed to search: %v\n", err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to search")
	}

	// Collect the search results
	results, err := pgx.CollectRows(rows, pgx.RowToStructByName[db.Project])

	// Handle errors during the search result collection
	if err != nil {
		log.Errorf("failed to collect search: %v\n", err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to collect search")
	}

	// Return the search results
	return c.JSON(http.StatusOK, SearchResults{
		Time:    time.Since(startTime).Seconds(),
		Count:   len(results),
		Results: results,
	})
}

func search(c echo.Context) error {
	// Get the query parameter from the request.
	var query = c.QueryParam("q")

	// Establish a connection to the database.
	var conn = db.EstablishConnection()

	// Keep track of the start time of the search.
	var startTime = time.Now()

	// Search the projects table for projects that match the query.
	var rows, err = conn.SearchProjects(query)

	// Define the structure of the search results.
	type SearchResults struct {
		Time    float64      `json:"time"` // Search time in seconds
		Count   int          `json:"count"`
		Results []db.Project `json:"results"`
	}

	// Handle errors during the search.
	if err != nil {
		// If no rows are found, return empty results.
		if err == pgx.ErrNoRows {
			return c.JSON(http.StatusOK, SearchResults{
				Time:    time.Since(startTime).Seconds(),
				Count:   0,
				Results: make([]db.Project, 0),
			})
		}

		// Log and return an error if the search fails.
		log.Errorf("failed to search: %v\n", err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to search")
	}

	// Collect the search results.
	results, err := pgx.CollectRows(rows, pgx.RowToStructByName[db.Project])

	// Handle errors during the search result collection.
	if err != nil {
		log.Errorf("failed to collect search: %v\n", err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to collect search")
	}

	// Return the search results.
	return c.JSON(http.StatusOK, SearchResults{
		Time:    time.Since(startTime).Seconds(),
		Count:   len(results),
		Results: results,
	})
}

func publishProject(c echo.Context) error {
	var id = c.Param("id")
	var rawToken = c.Request().Header.Get(echo.HeaderAuthorization)

	var conn = db.EstablishConnection()

	validToken, token := utils.ValidateToken(rawToken)

	if !validToken {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
	}

	project, err := conn.GetProjectByID(id)
	if err != nil {

		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no project with that id found")
		}

		log.Errorf("failed to fetch project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project")
	}

	isOwner, err := utils.IsUserProjectOwner(project, token, validToken)

	if err != nil {
		log.Errorf("failed to fetch project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project")
	}

	if !isOwner {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	if project.Status == "draft" {

		tx, err := conn.Db.Begin(context.Background())

		if err != nil {
			log.Errorf("failed to begin transaction: %v\n", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to publish project")
		}

		err = conn.UpdateProjectStatus(tx, id, "pending")

		if err != nil {
			newErr := tx.Rollback(context.Background())

			if newErr != nil {
				log.Errorf("failed to rollback: %v\n", newErr)
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to publish project")
			}

			log.Errorf("failed to update project status: %v\n", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to publish project")
		}

		if err = tx.Commit(context.Background()); err != nil {

			newErr := tx.Rollback(context.Background())

			if newErr != nil {
				log.Errorf("failed to rollback: %v\n", newErr)
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to publish project")
			}

			log.Errorf("failed to commit transaction: %v\n", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to publish project")
		}

	} else {
		return echo.NewHTTPError(http.StatusBadRequest, "project is already published")
	}

	return c.String(http.StatusOK, "project published")
}

func draftProject(c echo.Context) error {
	var id = c.Param("id")
	var rawToken = c.Request().Header.Get(echo.HeaderAuthorization)

	var conn = db.EstablishConnection()

	validToken, token := utils.ValidateToken(rawToken)

	if !validToken {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
	}

	project, err := conn.GetProjectByID(id)
	if err != nil {

		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no project with that id found")
		}

		log.Errorf("failed to fetch project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project")
	}

	isOwner, err := utils.IsUserProjectOwner(project, token, validToken)

	if err != nil {
		log.Errorf("failed to fetch project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project")
	}

	if !isOwner {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	if project.Status == "live" || project.Status == "pending" {

		tx, err := conn.Db.Begin(context.Background())

		if err != nil {
			log.Errorf("failed to begin transaction: %v\n", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to publish project")
		}

		err = conn.UpdateProjectStatus(tx, id, "draft")

		if err != nil {
			newErr := tx.Rollback(context.Background())

			if newErr != nil {
				log.Errorf("failed to rollback: %v\n", newErr)
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to publish project")
			}

			log.Errorf("failed to update project status: %v\n", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to publish project")
		}

		if err = tx.Commit(context.Background()); err != nil {

			newErr := tx.Rollback(context.Background())

			if newErr != nil {
				log.Errorf("failed to rollback: %v\n", newErr)
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to publish project")
			}

			log.Errorf("failed to commit transaction: %v\n", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to publish project")
		}

	} else {
		return echo.NewHTTPError(http.StatusBadRequest, "project is already in draft status")
	}

	return c.String(http.StatusOK, "project drafted")
}

func deleteProject(c echo.Context) error {

	var pid = c.Param("id")

	// Establish a connection to the database.
	var conn = db.EstablishConnection()

	// Get the project from the database.
	project, err := conn.GetProjectBySlug(pid)

	// If there was an error fetching the project, return an appropriate error.
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no project found")
		}
		log.Errorf("failed to fetch project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project")
	}

	// Get the token from the request headers.
	rawToken := c.Request().Header.Get(echo.HeaderAuthorization)
	validToken, token := utils.ValidateToken(rawToken)

	// If the token is invalid, return a forbidden error.
	if !validToken {
		return echo.NewHTTPError(http.StatusForbidden, "invalid token")
	}

	isOwner, err := utils.IsUserProjectOwner(project, token, validToken)

	if err != nil {
		log.Errorf("failed to fetch project owner: %v\n", err)
		return echo.NewHTTPError(http.StatusForbidden, "failed to fetch project owner")
	}

	if !isOwner {
		return echo.NewHTTPError(http.StatusForbidden, "not authorized")
	}

	tx, err := conn.Db.Begin(context.Background())

	if err != nil {
		log.Errorf("failed to begin transaction: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete project")
	}

	if err = conn.DeleteProject(tx, pid); err != nil {
		newErr := tx.Rollback(context.Background())

		if newErr != nil {
			log.Errorf("failed to rollback: %v\n", newErr)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete project")
		}

		log.Errorf("failed to delete project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete project")
	}

	if err = tx.Commit(context.Background()); err != nil {
		newErr := tx.Rollback(context.Background())

		if newErr != nil {
			log.Errorf("failed to rollback: %v\n", newErr)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete project")
		}

		log.Errorf("failed to commit transaction: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete project")
	}

	return c.String(http.StatusOK, "project deleted")
}

func featuredProjects(c echo.Context) error {
	conn := db.EstablishConnection()

	projects, err := conn.GetFeaturedProjects()

	if err != nil {
		log.Errorf("failed to fetch projects: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch projects")
	}

	return c.JSON(http.StatusOK, projects)
}

func RegisterProjectRoutes(e *echo.Echo) {
	e.GET("/projects", listProjects)
	e.GET("/projects/:id", getProjectById, utils.DevRateLimiter(100))
	e.GET("/projects/random", randomProject, utils.DevRateLimiter(100))
	e.GET("/projects/slug/:slug", getProjectBySlug, utils.DevRateLimiter(100))
	e.GET("/projects/search/full", ftsSearch)
	e.GET("/projects/search", search)
	e.GET("/projects/featured", featuredProjects, utils.DevRateLimiter(100))

	e.PUT("/projects/:id/publish", publishProject, utils.DevRateLimiter(100))
	e.PUT("/projects/:id/draft", draftProject, utils.DevRateLimiter(100))
	e.PUT("/projects/:id", updateProject, utils.DevRateLimiter(10))

	e.POST("/projects/create", createProject, utils.DevRateLimiter(10))

	e.DELETE("/projects/:id", deleteProject, utils.DevRateLimiter(10))
}
