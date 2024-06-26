package routes

import (
	"context"
	"fmt"
	utils "me/cobble/utils/db"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func getUserRoute(c echo.Context) error {
	var id = c.Param("id")
	var conn = utils.EstablishConnection()

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
	var user utils.User
	var conn = utils.EstablishConnection()

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

	user = utils.User{
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
		tx.Rollback(context.Background())
		fmt.Fprintf(os.Stderr, "failed to create user: %v\n", err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
	}

	// Create the user in the database.
	err = conn.CreateUser(tx, user)

	// Check for errors during the user creation.
	if err != nil {
		tx.Rollback(context.Background())
		fmt.Fprintf(os.Stderr, "failed to create user: %v\n", err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
	}

	// Commit the transaction.
	err = tx.Commit(context.Background())

	// Check for errors during the commit.
	if err != nil {
		tx.Rollback(context.Background())
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

func RegisterUserRoutes(e *echo.Echo) {
	e.GET("/users/:id", getUserRoute, middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(100)))
	e.POST("/users/create", createUser, middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(10)))
}
