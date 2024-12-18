package paging

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

const (
	DefaultSearchPageSize = 25
	MaxSearchPageSize     = 100
	FirstPageIndex        = 0
)

func GetPaginationModel(c echo.Context) (int, int, error) {
	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil {
		page = FirstPageIndex
	}
	if page < 0 {
		return 0, 0, echo.NewHTTPError(http.StatusBadRequest, "pages can't be negative")
	}

	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil {
		limit = DefaultSearchPageSize
	}

	if limit <= 0 {
		return 0, 0, echo.NewHTTPError(http.StatusBadRequest, "limit can't be negative or zero")
	}
	if limit > MaxSearchPageSize {
		return 0, 0, echo.NewHTTPError(http.StatusBadRequest, "limit can't be more then "+strconv.Itoa(MaxSearchPageSize))
	}

	offset := page * limit
	return limit, offset, nil
}
