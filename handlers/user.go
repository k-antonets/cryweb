package handlers

import (
	"github.com/lab7arriam/cryweb/models"
	"github.com/labstack/echo"
	"gopkg.in/mgo.v2/bson"
)

func (h *Handler) Login(c echo.Context) error {
	u := new(models.User)
	if err := c.Bind(u); err != nil {
		return err
	}

	if err := h.DB.DB(h.Database).
		C("users").Find(bson.M{"email": u.Email}).One(u); err != nil {

	}
	return echo.ErrBadRequest
}

func (h *Handler) Register(c echo.Context) error {
	return echo.ErrBadRequest
}
