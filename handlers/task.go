package handlers

import (
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/lab7arriam/cryweb/models"
	"github.com/labstack/echo/v4"
	"gopkg.in/mgo.v2/bson"
)

func (h *Handler) TasksList(c echo.Context) error {
	tool := c.Param("tool")
	userJwt := c.Get("user").(*jwt.Token)
	claims := userJwt.Claims.(*JwtUserClaims)
	user_id := claims.Email

	var tasks []*models.Task

	if err := h.DbTask().Find(bson.M{"tool": tool, "user_id": user_id}).Sort("-created").All(&tasks); err != nil {
		return h.indexAlert(c, http.StatusBadGateway, "failed to get list of tasks", "error")
	}

	return c.Render(http.StatusOK, "pages/tasks", echo.Map{
		"tasks":        tasks,
		"add_task_url": h.Route("tasks.add_form", tool),
	})
}

func (h *Handler) AddTask(c echo.Context) error {
	return nil
}
