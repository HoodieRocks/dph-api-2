package routes

import (
	"context"
	"fmt"
	"me/cobble/utils"
	"me/cobble/utils/db"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

func getUserRoute(c echo.Context) error {
	var id = c.Param("id")
	var conn = db.EstablishConnection()

	user, err := conn.GetUserById(id)

	if err != nil {

		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no user found")
		}

		fmt.Fprintf(os.Stderr, "failed to fetch user: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch user")
	}

	return c.JSON(http.StatusOK, user)
}

func createUser(c echo.Context) error {
	// Declare variables for the user and connection.
	var user db.User
	var conn = db.EstablishConnection()

	// Retrieve the username and password from the request form values.
	var username = c.FormValue("username")
	var password = c.FormValue("password")

	// Check if the username is already taken.
	var usernameTaken = conn.CheckForUsernameConflict(username)

	// Validate the password length.
	if len(password) < 8 {
		return echo.NewHTTPError(http.StatusBadRequest, "password too short!")
	}

	// Generate an Argon2id hash for the password.
	var passHash, _ = argon2id.CreateHash(password, argon2id.DefaultParams)

	// If the username is not taken, create a new user object.
	if usernameTaken {
		// Return an error if the username is already taken.
		return echo.NewHTTPError(http.StatusConflict, "a user with that name already exists")
	}

	user = db.User{
		Username: username,
		Role:     "default",
		Bio:      "A new user!",
		JoinDate: time.Now(),
		Password: passHash,
		Token:    utils.GenerateSecureToken(),
	}

	// Begin a transaction.
	var tx, err = conn.Db.Begin(context.Background())

	// Check for errors during the transaction.
	if err != nil {
		newErr := tx.Rollback(context.Background())

		if newErr != nil {
			fmt.Fprintf(os.Stderr, "failed to rollback transaction: %v\n", newErr)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
		}
		fmt.Fprintf(os.Stderr, "failed to create user: %v\n", err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
	}

	// Create the user in the database.
	err = conn.CreateUser(tx, user)

	// Check for errors during the user creation.
	if err != nil {
		newErr := tx.Rollback(context.Background())

		if newErr != nil {
			fmt.Fprintf(os.Stderr, "failed to rollback transaction: %v\n", newErr)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
		}
		fmt.Fprintf(os.Stderr, "failed to create user: %v\n", err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
	}

	// Commit the transaction.
	err = tx.Commit(context.Background())

	// Check for errors during the commit.
	if err != nil {
		newErr := tx.Rollback(context.Background())

		if newErr != nil {
			fmt.Fprintf(os.Stderr, "failed to rollback transaction: %v\n", newErr)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
		}
		fmt.Fprintf(os.Stderr, "failed to create user: %v\n", err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
	}

	// Set a cookie with the user's token.
	c.SetCookie(&http.Cookie{
		Name:    "DPH_TOKEN",
		Value:   user.Token,
		Expires: time.Now().AddDate(0, 1, 0),
	})

	// Return the created user as a JSON response.
	return c.JSON(http.StatusCreated, user)
}

func getSelf(c echo.Context) error {
	var rawToken = c.Request().Header.Get(echo.HeaderAuthorization)
	var validToken, token = utils.ValidateToken(rawToken)

	if !validToken {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
	}

	var conn = db.EstablishConnection()
	var user, err = conn.GetUserByToken(*token)

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch user: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch user")
	}

	return c.JSON(http.StatusOK, user)
}

func logOut(c echo.Context) error {
	c.SetCookie(&http.Cookie{
		Name:    "DPH_TOKEN",
		Value:   "",
		Expires: time.Now().AddDate(0, 0, -1),
	})

	return c.NoContent(http.StatusNoContent)
}

func getProjectsByUser(c echo.Context) error {

	var id = c.Param("id")
	var conn = db.EstablishConnection()

	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	if strings.ToLower(id) == "me" {
		var rawToken = c.Request().Header.Get(echo.HeaderAuthorization)
		var validToken, token = utils.ValidateToken(rawToken)

		if !validToken {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}

		var conn = db.EstablishConnection()
		var user, err = conn.GetUserByToken(*token)

		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to fetch user: %v\n", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch user")
		}

		id = user.ID
	}

	projects, err := conn.GetAllProjectsByAuthor(id)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch projects")
	}

	return c.JSON(http.StatusOK, projects)
}

type StaffResponse struct {
	Count int       `json:"count"`
	Users []db.User `json:"users"`
}

func getStaff(c echo.Context) error {
	var conn = db.EstablishConnection()
	var role = c.Param("role")

	if role == "" {
		role = "helper"
	}

	if role != "helper" && role != "admin" && role != "moderator" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role")
	}

	staff, err := conn.GetAllInRole(role)

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch staff: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch staff")
	}

	return c.JSON(http.StatusOK, StaffResponse{
		Count: len(staff),
		Users: staff,
	})
}

func RegisterUserRoutes(e *echo.Echo) {
	e.GET("/users/:id", getUserRoute, utils.DevRateLimiter(100))
	e.GET("/users/me", getSelf, utils.DevRateLimiter(100))
	e.GET("/users/me/logout", logOut, utils.DevRateLimiter(100))
	e.GET("/users/projects/:id", getProjectsByUser, utils.DevRateLimiter(100))
	e.GET("/users/staff/:role", getStaff, utils.DevRateLimiter(100))
	e.POST("/users/create", createUser, utils.DevRateLimiter(10))
}
