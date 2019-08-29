package main

import (
	"net/http"

	"github.com/labstack/echo"
)

func IndexHndl(c echo.Context) error {
	return c.Render(http.StatusOK, "pages/index", echo.Map{})
}

func LoginHndl(c echo.Context) error {
	return c.Render(http.StatusOK, "pages/login", echo.Map{})
}
