package routes

import (
	"context"
	"fmt"
	d_errors "me/cobble/errors"
	files "me/cobble/utils"
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

// getVersionOnProject retrieves a version of a project from the database.
// It checks if the version exists and if the user has permission to access it.
// The version is identified by the project ID and the index of its creation.
// If the version is not found, it returns a 404 error.
// If the project is not found, it returns a 404 error.
// If the version ID is not a number or "latest", it returns a 400 error.
// If the token is invalid or expired, it returns a 403 error.
// If the user does not have permission to access the project, it returns a 403 error.
// If the project is in an illegal state, it returns a 500 error.
// It returns the version as JSON if the project is live, or if the user is the project owner.
func getVersionOnProject(c echo.Context) error {
	// Extract the project ID and the index of the version from the parameters.
	pid := c.Param("pid")
	rawIdx := c.Param("idx")

	// Convert the index to an integer.
	idx, err := strconv.Atoi(rawIdx)

	// If the index is not "latest" and cannot be converted to an integer, return a 400 error.
	if rawIdx != "latest" && err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch version: %v\n", err)
		return echo.NewHTTPError(http.StatusBadRequest, "version ID must be a number or latest")
	}

	// If the index is "latest", set it to 0.
	if rawIdx == "latest" {
		idx = 0
	}

	// Establish a connection to the database.
	var conn = utils.EstablishConnection()

	// Retrieve the project from the database.
	project, err := conn.GetProjectByID(pid)

	// If the project is not found, return a 404 error.
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no project with that id found")
		}

		fmt.Fprintf(os.Stderr, "failed to fetch version parent: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch version parent")
	}

	// Retrieve the version from the database.
	version, err := conn.GetVersionByCreation(pid, idx)

	// If the version is not found or could not be retrieved, return a 404 error.
	if err != nil || version == nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no version found")
		}

		fmt.Fprintf(os.Stderr, "failed to fetch version: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch version")
	}

	// Retrieve the token from the request header.
	rawToken := c.Request().Header.Get(echo.HeaderAuthorization)

	// Validate the token.
	validToken, token := utils.TokenValidate(rawToken)

	// Check if the project is live or if the project is in draft mode, then check if the user is the project owner.
	switch project.Status {
	case "live":
		// If the project is live, return the version as JSON.
		return c.JSON(http.StatusOK, version)
	case "draft":
		// Check if the user is the project owner.
		isOwner, err := utils.IsUserProjectOwner(project, token, validToken)

		if err != nil {
			return err
		}

		// Check if the user is the project owner.
		if isOwner {
			// If the user is the project owner, return the version as JSON.
			return c.JSON(http.StatusOK, version)
		} else {
			// If the user does not have permission to access the project, return a 403 error.
			return echo.NewHTTPError(http.StatusForbidden, "you can not access other's private projects")
		}
	case "pending":
		// Check if the user is the project owner.
		isOwner, err := utils.IsUserProjectOwner(project, token, validToken)

		if err != nil {
			return err
		}

		// Check if the user is the project owner.
		if isOwner {
			// If the user is the project owner, return the version as JSON.
			return c.JSON(http.StatusOK, version)
		} else {
			// If the user does not have permission to access the project, return a 403 error.
			return echo.NewHTTPError(http.StatusForbidden, "you can not access other's private projects")
		}
	default:
		// If the project is in an illegal state, return a 500 error.
		return echo.NewHTTPError(http.StatusInternalServerError, "illegal project state")
	}
}

// createVersion handles the creation of a new version for a project.
// It expects an authorization token in the request header, as well as
// form values for the title, description, version_code, supports, and
// download files. It returns a JSON representation of the created version
// or an error if any occurred.
func createVersion(c echo.Context) error {
	// Retrieve the authorization token from the request header.
	var rawToken = c.Request().Header.Get("Authorization")

	// Retrieve the project ID and form values from the request.
	var pid = c.Param("pid")
	var title = c.FormValue("title")
	var description = c.FormValue("description")
	var versionCode = c.FormValue("version_code")
	var supports = strings.Split(c.FormValue("supports"), ",")
	download, err := c.FormFile("download")

	// If the download file is missing, return a 400 error.
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "download file is required")
	}

	// Retrieve the resource pack file from the request.
	rpDownload, _ := c.FormFile("rpDownload")

	// Establish a database connection.
	conn := utils.EstablishConnection()

	// Validate the authorization token.
	validToken, token := utils.TokenValidate(rawToken)

	// If the token is invalid, return a 400 error.
	if !validToken || token == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "malformed token")
	}

	// Retrieve the project from the database using the project ID.
	project, err := conn.GetProjectByID(pid)

	// If the project could not be retrieved, return a 404 error.
	if err != nil {

		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no project with that id found")
		}

		fmt.Fprintf(os.Stderr, "failed to fetch project: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project")
	}

	// Retrieve the user from the token.
	user, err := conn.GetUserByToken(*token)

	// If the user could not be retrieved, return a 403 error.
	if err == pgx.ErrNoRows {
		return echo.NewHTTPError(http.StatusForbidden, "invalid token")
	}

	// Check if the user is the project owner.
	if project.Author != user.ID {
		return echo.NewHTTPError(http.StatusForbidden, "you can not access other's private projects")
	}

	// Upload the version file to the server.
	downloadLink, err := files.UploadVersionFile(download)

	// If the upload failed, return a 400 error.
	if err != nil {

		if err == d_errors.ErrFileTooLarge {
			return echo.NewHTTPError(http.StatusBadRequest, "version file is too big")
		}

		if err == d_errors.ErrFileBadExtension {
			return echo.NewHTTPError(http.StatusBadRequest, "bad version file extension")
		}

		fmt.Fprintf(os.Stderr, "failed to upload file: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to upload file")
	}

	// Create a new version object.
	var version = utils.Version{
		Title:        title,
		Description:  description,
		Creation:     time.Now(),
		Downloads:    0,
		DownloadLink: downloadLink,
		Supports:     supports,
		Project:      project.ID,
		VersionCode:  versionCode,
	}

	// If a resource pack file was provided, upload it to the server.
	if rpDownload != nil {

		rpDownloadLink, err := files.UploadResourcePackFile(rpDownload)

		// If the upload failed, return a 400 error.
		if err != nil {

			if err == d_errors.ErrFileTooLarge {
				return echo.NewHTTPError(http.StatusBadRequest, "resource pack file is too big")
			}

			if err == d_errors.ErrFileBadExtension {
				return echo.NewHTTPError(http.StatusBadRequest, "bad resource file extension")
			}

			fmt.Fprintf(os.Stderr, "failed to upload file: %v\n", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to upload file")
		}

		// Set the resource pack download link in the version object.
		version.RpDownload = &rpDownloadLink
	}

	// Start a transaction to create the version in the database.
	tx, err := conn.Db.Begin(context.Background())

	// If the transaction failed, rollback and return a 500 error.
	if err != nil {
		newErr := tx.Rollback(context.Background())
		
		if newErr != nil {
			fmt.Fprintf(os.Stderr, "failed to rollback transaction: %v\n", newErr)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
		}
		fmt.Fprintf(os.Stderr, "failed to create version: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	// Create the version in the database.
	err = conn.CreateVersion(tx, project.ID, version)

	// If the creation failed, rollback and return a 500 error.
	if err != nil {
		newErr := tx.Rollback(context.Background())
		
		if newErr != nil {
			fmt.Fprintf(os.Stderr, "failed to rollback transaction: %v\n", newErr)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
		}
		fmt.Fprintf(os.Stderr, "failed to create version: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	// Commit the transaction.
	err = tx.Commit(context.Background())

	// If the commit failed, rollback and return a 500 error.
	if err != nil {
		newErr := tx.Rollback(context.Background())
		
		if newErr != nil {
			fmt.Fprintf(os.Stderr, "failed to rollback transaction: %v\n", newErr)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
		}
		fmt.Fprintf(os.Stderr, "failed to create version: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	// Return the created version as JSON.
	return c.JSON(http.StatusCreated, version)
}

// listVersions returns all versions of a project. If the project is in a draft state,
// it requires the user's token be valid and the owner of the project.
func listVersions(c echo.Context) error {
	// Get the project ID from the request parameters.
	pid := c.Param("pid")

	// Establish a connection to the database.
	var conn = utils.EstablishConnection()

	// Get the project from the database.
	project, err := conn.GetProjectByID(pid)

	// If the project was not found, return a 404 error.
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no project with that id found")
		}

		// If there was an error fetching the project, return a 500 error.
		fmt.Fprintf(os.Stderr, "failed to fetch version parent: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch version parent")
	}

	// Get all versions of the project from the database.
	versions, err := conn.GetAllProjectVersions(pid)

	// If there were no versions found, return a 404 error.
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no version found")
		}

		// If there was an error fetching the versions, return a 500 error.
		fmt.Fprintf(os.Stderr, "failed to fetch version: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch version")
	}

	// Get the token from the request headers.
	rawToken := c.Request().Header.Get(echo.HeaderAuthorization)

	// Validate the token.
	validToken, token := utils.TokenValidate(rawToken)

	// Check the status of the project.
	switch project.Status {
	case "live":
		// If the project is live, return the versions.
		return c.JSON(http.StatusOK, versions)

	case "draft":
		// Check if the user is the project owner.
		isOwner, err := utils.IsUserProjectOwner(project, token, validToken)

		if err != nil {
			return err
		}

		// Check if the user is the project owner.
		if isOwner {
			return c.JSON(http.StatusOK, versions)
		} else {
			// If the user is not the owner, return a 403 error.
			return echo.NewHTTPError(http.StatusForbidden, "you can not access other's private projects")
		}
	case "pending":
		// Check if the user is the project owner.
		isOwner, err := utils.IsUserProjectOwner(project, token, validToken)

		if err != nil {
			return err
		}

		// Check if the user is the project owner.
		if isOwner {
			// If the user is the project owner, return the version as JSON.
			return c.JSON(http.StatusOK, versions)
		} else {
			// If the user does not have permission to access the project, return a 403 error.
			return echo.NewHTTPError(http.StatusForbidden, "you can not access other's private projects")
		}
	default:
		// If the project is in an illegal state, return a 500 error.
		return echo.NewHTTPError(http.StatusInternalServerError, "illegal project state")
	}
}

func downloadVersion(c echo.Context) error {
	pid := c.Param("pid")

	var conn = utils.EstablishConnection()

	project, err := conn.GetProjectByID(pid)

	if err != nil {

		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no project with that id found")
		}

		fmt.Fprintf(os.Stderr, "failed to fetch version parent: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch version parent")
	}

	idx, err := strconv.Atoi(c.Param("vid"))
	if err != nil {

		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no version found")
		}

		fmt.Fprintf(os.Stderr, "failed to fetch version: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch version")
	}

	// Get the token from the request headers.
	rawToken := c.Request().Header.Get(echo.HeaderAuthorization)
	validToken, token := utils.TokenValidate(rawToken)

	// Get the version from the database.
	version, err := conn.GetVersionByCreation(pid, idx)

	// If there was an error fetching the version, return an appropriate error.
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no version found")
		}
		fmt.Fprintf(os.Stderr, "failed to fetch version: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch version")
	}

	// Check the status of the project.
	switch project.Status {
	case "live":
		tx, err := conn.Db.Begin(context.Background())

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to begin transaction")
		}

		err = conn.UpdateProjectDownloads(tx, project.ID, project.Downloads+1)

		if err != nil {
			newErr := tx.Rollback(context.Background())
		
			if newErr != nil {
				fmt.Fprintf(os.Stderr, "failed to rollback transaction: %v\n", newErr)
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
			}

			fmt.Fprintf(os.Stderr, "failed to update downloads: %v\n", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to update project downloads")
		}

		err = tx.Commit(context.Background())

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to commit transaction")
		}

		return c.File(version.DownloadLink)
	case "draft":

		// Check if the user is the owner of the project.
		isOwner, err := utils.IsUserProjectOwner(project, token, validToken)

		if err != nil {
			return err
		}

		// Check if the user is the owner of the project.
		if isOwner {
			return c.File(version.DownloadLink)
		} else {
			// If the user is not the owner, return a forbidden error.
			return echo.NewHTTPError(http.StatusForbidden, "you can not access other's private projects")
		}
	case "pending":

		// Check if the user is the owner of the project.
		isOwner, err := utils.IsUserProjectOwner(project, token, validToken)

		if err != nil {
			return err
		}

		// Check if the user is the owner of the project.
		if isOwner {
			return c.File(version.DownloadLink)
		} else {
			// If the user is not the owner, return a forbidden error.
			return echo.NewHTTPError(http.StatusForbidden, "you can not access other's private projects")
		}
	default:
		// If the project is in an illegal state, return an internal server error.
		return echo.NewHTTPError(http.StatusInternalServerError, "illegal project state")
	}
}

func RegisterVersionRoutes(e *echo.Echo) {
	e.GET("/projects/:pid/versions/:idx", getVersionOnProject, middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(10)))
	e.GET("/projects/:pid/versions", listVersions, middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(10)))
	e.POST("/projects/:pid/versions/create", createVersion, middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(10)))
	e.GET("/projects/:pid/versions/:idx/download", downloadVersion, middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(10)))
}
