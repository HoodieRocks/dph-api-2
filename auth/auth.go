package auth

import (
	"errors"
	"github.com/HoodieRocks/dph-api-2/utils/db"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"net/http"
	"slices"
	"strings"
)

const userDataContextKey = "HoodieRocks/dph-api-2/user"

const (
	ModeratorRole = "moderator"
	AdminRole     = "admin"
)

func Token2UserContext(next echo.HandlerFunc) echo.HandlerFunc {

	return func(c echo.Context) error {

		var rawToken = c.Request().Header.Get(echo.HeaderAuthorization)
		if rawToken == "" {
			return next(c)
		}

		var validToken, token = validateToken(rawToken)
		if !validToken || token == nil {
			return echo.NewHTTPError(http.StatusBadRequest, "malformed token")
		}
		var conn = db.EstablishConnection()
		var user, err = conn.GetUserByToken(*token)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				log.Errorf("no user is assigned to this token")
				return echo.NewHTTPError(http.StatusForbidden, "invalid token")
			}
			log.Errorf("failed to fetch project owner: %v\n", err)
			return echo.NewHTTPError(http.StatusForbidden, "invalid token")
		}

		c.Set(userDataContextKey, user)

		return next(c)
	}
}

func validateToken(token string) (bool, *string) {
	var tokenParts = strings.Split(token, " ")
	if len(tokenParts) < 2 {
		return false, nil
	}
	return tokenParts[0] == "Bearer" && len(tokenParts[1]) > 8, &tokenParts[1]
}

func AllowRoles(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		//bypass mode
		//TODO get a second opinion
		if len(roles) == 0 {
			return func(c echo.Context) error {
				return next(c)
			}
		}

		return func(c echo.Context) error {
			user, err := GetContextUser(c)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "you need to login")
			}
			if !slices.Contains(roles, user.Role) {
				return echo.NewHTTPError(http.StatusForbidden, "you do not have permission to view this")
			}
			return next(c)
		}
	}
}

func GetContextUser(c echo.Context) (db.User, error) {
	user, ok := c.Get(userDataContextKey).(db.User)
	if !ok {
		return db.User{}, errors.New("no user in context")
	}
	return user, nil
}
