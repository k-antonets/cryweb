package handlers

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func (h *Handler) ToolMain(c echo.Context) error {
	u, l := h.checkLogged(c)
	if c.Param("tool") != "cry_processor" {
		return c.Render(http.StatusNotFound, "pages/index", echo.Map{
			"tool_name":    c.Param("tool"),
			"login_url":    h.Route("user.login"),
			"register_url": h.Route("user.register"),
			"notification": "Tool is not found",
			"alert_type":   "error",
			"logged":       l,
			"user":         u,
		})
	}
	tool_name := "Cry Processor" // TODO: should be replaced by adding tools to db
	return c.Render(http.StatusOK, "pages/index", echo.Map{
		"tool_name":    tool_name,
		"login_url":    h.Route("user.login"),
		"register_url": h.Route("user.register"),
		"logged":       l,
		"user":         u,
	})
}
