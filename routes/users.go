package routes

import (
	"context"
	"github.com/HoodieRocks/dph-api-2/auth"
	"net/http"
	"strings"
	"time"

	"github.com/HoodieRocks/dph-api-2/utils"
	"github.com/HoodieRocks/dph-api-2/utils/db"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

const MinPassLen = 8
const DPHToken = "DPH_TOKEN"

func getUserRoute(c echo.Context) error {
	var id = c.Param("id")
	var conn = db.EstablishConnection()

	user, err := conn.GetUserById(id)

	if err != nil {

		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "no user found")
		}

		log.Errorf("failed to fetch user: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch user")
	}

	return c.JSON(http.StatusOK, user)
}

func createUser(c echo.Context) error {
	// Declare variables for the user and connection.
	var user db.User

	// Retrieve the username and password from the request form values.
	var username = c.FormValue("username")
	var password = c.FormValue("password")

	// Validate the password length.
	//PZ - introduced constant instead of magic-number
	if len(password) < MinPassLen {
		return echo.NewHTTPError(http.StatusBadRequest, "password too short!")
	}

	// Generate an Argon2id hash for the password.
	var passHash, hashErr = argon2id.CreateHash(password, argon2id.DefaultParams)

	//PZ - handled error in secure generation
	if hashErr != nil {
		log.Errorf("failed to generate enough entropy to secure new password hash %v\n", hashErr)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
	}

	//PZ - moved database operations lower in check, to prevent connection exhaustion
	// Check if the username is already taken.
	var conn = db.EstablishConnection()

	var usernameTaken = conn.CheckForUsernameConflict(username)
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
		Token:    auth.GenerateSecureToken(),
	}

	// Begin a transaction.
	var tx, err = conn.Db.Begin(context.Background())

	// Check for errors during the transaction.
	err2, failed := handleErrorInTransaction(err, tx)
	if failed {
		return err2
	}

	// Create the user in the database.
	err = conn.CreateUser(tx, user)

	// Check for errors during the user creation.
	err2, failed = handleErrorInTransaction(err, tx)
	if failed {
		return err2
	}

	// Commit the transaction.
	err = tx.Commit(context.Background())

	// Check for errors during the commit.
	err2, failed = handleErrorInTransaction(err, tx)
	if failed {
		return err2
	}

	// Set a cookie with the user's token.
	//TODO where are they used?
	c.SetCookie(&http.Cookie{
		Name:    DPHToken,
		Value:   user.Token,
		Expires: time.Now().AddDate(0, 1, 0),
	})

	// Return the created user as a JSON response.
	return c.JSON(http.StatusCreated, user)
}

func handleErrorInTransaction(err error, tx pgx.Tx) (error, bool) {
	if err != nil {
		newErr := tx.Rollback(context.Background())

		if newErr != nil {
			log.Errorf("failed to rollback transaction: %v\n", newErr)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project"), true
		}
		log.Errorf("failed to create user: %v\n", err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user"), true
	}
	return nil, false
}

func getSelf(c echo.Context) error {
	user, err := auth.GetContextUser(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	return c.JSON(http.StatusOK, user)
}

func logOut(c echo.Context) error {

	//TODO rewrite on proper sessions, as this is just like password in cookie, as provides same level of protection
	c.SetCookie(&http.Cookie{
		Name:    DPHToken,
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
		var user, err = auth.GetContextUser(c)

		if err != nil {
			log.Errorf("failed to fetch user: %v\n", err)
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
		log.Errorf("failed to fetch staff: %v\n", err)
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

	e.POST("/users/create", createUser, utils.DevRateLimiter(1))
}
