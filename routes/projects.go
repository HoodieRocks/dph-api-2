package routes

import (
	"context"
	"fmt"
	utils "me/cobble/utils/db"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// listProjects retrieves a list of projects from the database based on the provided query parameters.
// It returns a JSON representation of the projects.
// If the query parameters are invalid, it returns an internal server error.
func listProjects(c echo.Context) error {
	// Define the structure of the search results.
	type SearchResults struct {
		Time    float64         `json:"time"` // Search time in seconds
		Count   int             `json:"count"`
		Results []utils.Project `json:"results"`
	}

	// Establish a connection to the database
	var conn = utils.EstablishConnection()

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
		fmt.Fprintf(os.Stderr, "failed to fetch projects: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch projects")
	}

	// Return the projects as JSON
	return c.JSON(http.StatusOK, SearchResults{
		Time:    time.Since(startTime).Seconds(),
		Count:   len(results),
		Results: results,
	})
}

// getProjectById retrieves a project from the database based on the provided ID.
// It checks the project's status and returns the project if it is live,
// or if the user is the owner of the project and has a valid token.
// If the project is draft and the user is the owner but does not have a valid token,
// it returns a forbidden error.
// If the project is draft and the user is not the owner, it returns a forbidden error.
// If the project is in an illegal state, it returns an internal server error.
func getProjectById(c echo.Context) error {
	// Get the project ID from the URL parameter.
	id := c.Param("id")

	// Establish a connection to the database.
	var conn = utils.EstablishConnection()

	// Get the project from the database.
	project, err := conn.GetProjectByID(id)

	// If there was an error fetching the project, return an appropriate error.
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no project found")
		}
		fmt.Fprintf(os.Stderr, "failed to fetch project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project")
	}

	// Get the token from the request headers.
	rawToken := c.Request().Header.Get(echo.HeaderAuthorization)
	validToken, token := utils.TokenValidate(rawToken)

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

		conn := utils.EstablishConnection()

		user, err := conn.GetUserByToken(*token)

		if err != nil {
			if err == pgx.ErrNoRows {
				return echo.NewHTTPError(http.StatusNotFound, "no user is assigned to this token")
			}
			fmt.Fprintf(os.Stderr, "failed to fetch project owner: %v\n", err)
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

// getProjectBySlug retrieves a project from the database based on the provided ID.
// It checks the project's status and returns the project if it is live,
// or if the user is the owner of the project and has a valid token.
// If the project is draft and the user is the owner but does not have a valid token,
// it returns a forbidden error.
// If the project is draft and the user is not the owner, it returns a forbidden error.
// If the project is in an illegal state, it returns an internal server error.
func getProjectBySlug(c echo.Context) error {
	// Get the project ID from the URL parameter.
	id := c.Param("slug")

	// Establish a connection to the database.
	var conn = utils.EstablishConnection()

	// Get the project from the database.
	project, err := conn.GetProjectBySlug(id)

	// If there was an error fetching the project, return an appropriate error.
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no project found")
		}
		fmt.Fprintf(os.Stderr, "failed to fetch project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project")
	}

	// Get the token from the request headers.
	rawToken := c.Request().Header.Get(echo.HeaderAuthorization)
	validToken, token := utils.TokenValidate(rawToken)

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

		conn := utils.EstablishConnection()

		user, err := conn.GetUserByToken(*token)

		if err != nil {
			if err == pgx.ErrNoRows {
				return echo.NewHTTPError(http.StatusNotFound, "no user is assigned to this token")
			}
			fmt.Fprintf(os.Stderr, "failed to fetch project owner: %v\n", err)
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

// createProject handles the creation of a new project.
// It expects a form containing the project's title, slug, description, body, and category.
// It also expects a valid token in the Authorization header.
// It returns the created project in JSON format.
func createProject(c echo.Context) error {
	// Get the token from the Authorization header
	rawToken := c.Request().Header.Get("Authorization")

	// Get the form values
	title := c.FormValue("title")
	slug := c.FormValue("slug")
	description := c.FormValue("description")
	body := c.FormValue("body")
	// Split the category string by commas and convert it to a slice of strings
	category := strings.Split(c.FormValue("category"), ",")

	// Establish a connection to the database
	conn := utils.EstablishConnection()

	// Validate the token
	validToken, token := utils.TokenValidate(rawToken)

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

	// Create the project struct
	project := utils.Project{
		Title:       title,
		Slug:        slug,
		Author:      user.ID,
		Description: description,
		Body:        body,
		Creation:    time.Now(),
		Updated:     time.Now(),
		Category:    category,
	}

	// Start a transaction
	tx, err := conn.Db.Begin(context.Background())

	// If there was an error starting the transaction, rollback and return an internal server error
	if err != nil {
		tx.Rollback(context.Background())
		fmt.Fprintf(os.Stderr, "failed to create project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	// Create the project in the database
	err = conn.CreateProject(tx, project)

	// If there was an error creating the project, rollback and return an internal server error
	if err != nil {
		tx.Rollback(context.Background())
		fmt.Fprintf(os.Stderr, "failed to create project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	// Commit the transaction
	err = tx.Commit(context.Background())

	// If there was an error committing the transaction, rollback and return an internal server error
	if err != nil {
		tx.Rollback(context.Background())
		fmt.Fprintf(os.Stderr, "failed to create project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	// Return the created project in JSON format
	return c.JSON(http.StatusOK, project)
}

// changeProjectStatus handles the request to change the status of a project.
// It expects an ID and a status in the request parameters and a valid token in the authorization header.
// Only users with the role "moderator" or "admin" are allowed to change the project status.
//
// This function returns an HTTP error if the token is invalid or the user does not have permission,
// or if the project is not found.
// It returns a success message if the status is successfully updated.
func changeProjectStatus(c echo.Context) error {
	// Get the project ID and status from the request parameters
	id := c.Param("id")
	status := c.FormValue("status")

	if status == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing status")
	}

	// Establish a connection to the database
	conn := utils.EstablishConnection()

	// Get the token from the authorization header
	rawToken := c.Request().Header.Get(echo.HeaderAuthorization)

	// Validate the token
	validToken, token := utils.TokenValidate(rawToken)
	if !validToken || token == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "malformed token")
	}

	// Get the moderator or admin user from the token
	mod, err := conn.GetUserByToken(*token)
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusForbidden, "invalid token")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch privileged user information")
	}

	// Check if the user has permission to change the project status
	if mod.Role != "moderator" && mod.Role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "you do not have permission to change the project status")
	}

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
		tx.Rollback(context.Background())
		fmt.Fprintf(os.Stderr, "failed to create project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	// Update the project status
	err = conn.UpdateProjectStatus(tx, project.ID, status)
	if err != nil {
		tx.Rollback(context.Background())
		fmt.Fprintf(os.Stderr, "failed to create project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	// Commit the transaction
	err = tx.Commit(context.Background())

	// If the commit failed, rollback and return a 500 error.
	if err != nil {
		tx.Rollback(context.Background())
		fmt.Fprintf(os.Stderr, "failed to create project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	// Return a success message
	return c.String(http.StatusOK, "status updated")
}

// ftsSearch handles the FTS search API endpoint.
// It searches for projects using the full-text search index.
//
// It expects a "q" query parameter, which is the search query.
// It returns a SearchResults struct containing the search results.
// The struct contains the search time, the count of results, and the results themselves.
func ftsSearch(c echo.Context) error {
	// Get the search query from the request parameters
	query := c.QueryParam("q")

	// Establish a connection to the database
	conn := utils.EstablishConnection()

	// Record the start time of the search
	startTime := time.Now()

	// Perform the full-text search
	rows, err := conn.FTSSearchProjects(query)

	// Define the search results structure
	type SearchResults struct {
		Time    float64         `json:"time"` // Search time in seconds
		Count   int             `json:"count"`
		Results []utils.Project `json:"results"`
	}

	// Handle errors during the search
	if err != nil {
		if err == pgx.ErrNoRows {
			// Return empty results if no rows are found
			return c.JSON(http.StatusOK, SearchResults{
				Time:    time.Since(startTime).Seconds(),
				Count:   0,
				Results: make([]utils.Project, 0),
			})
		}

		// Log and return an error if the search fails
		fmt.Fprintf(os.Stderr, "failed to search: %v\n", err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to search")
	}

	// Collect the search results
	results, err := pgx.CollectRows(rows, pgx.RowToStructByName[utils.Project])

	// Handle errors during the search result collection
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to collect search: %v\n", err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to collect search")
	}

	// Return the search results
	return c.JSON(http.StatusOK, SearchResults{
		Time:    time.Since(startTime).Seconds(),
		Count:   len(results),
		Results: results,
	})
}

// search handles the "/projects/search" route and performs a search on the projects table.
// It takes a query parameter "q" which is used to search for projects.
// It returns a JSON response with the search results.
func search(c echo.Context) error {
	// Get the query parameter from the request.
	var query = c.QueryParam("q")

	// Establish a connection to the database.
	var conn = utils.EstablishConnection()

	// Keep track of the start time of the search.
	var startTime = time.Now()

	// Search the projects table for projects that match the query.
	var rows, err = conn.SearchProjects(query)

	// Define the structure of the search results.
	type SearchResults struct {
		Time    float64         `json:"time"` // Search time in seconds
		Count   int             `json:"count"`
		Results []utils.Project `json:"results"`
	}

	// Handle errors during the search.
	if err != nil {
		// If no rows are found, return empty results.
		if err == pgx.ErrNoRows {
			return c.JSON(http.StatusOK, SearchResults{
				Time:    time.Since(startTime).Seconds(),
				Count:   0,
				Results: make([]utils.Project, 0),
			})
		}

		// Log and return an error if the search fails.
		fmt.Fprintf(os.Stderr, "failed to search: %v\n", err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to search")
	}

	// Collect the search results.
	results, err := pgx.CollectRows(rows, pgx.RowToStructByName[utils.Project])

	// Handle errors during the search result collection.
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to collect search: %v\n", err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to collect search")
	}

	// Return the search results.
	return c.JSON(http.StatusOK, SearchResults{
		Time:    time.Since(startTime).Seconds(),
		Count:   len(results),
		Results: results,
	})
}

func RegisterProjectRoutes(e *echo.Echo) {
	e.GET("/projects", listProjects)
	e.GET("/projects/:id", getProjectById, middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(100)))
	e.GET("/projects/slug/:slug", getProjectBySlug, middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(100)))
	e.GET("/projects/search/full", ftsSearch)
	e.GET("/projects/search", search)
	e.POST("/projects/create", createProject, middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(10)))
	e.PUT("/projects/:id/status", changeProjectStatus, middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(10)))
}
