package handlers

import (
	"github.com/lab7arriam/cryweb/providers"
	"gopkg.in/mgo.v2"
)

type Handler struct {
	DB       *mgo.Session
	Database string
	Key      string
	ES       *providers.EmailSender
	Url      string
}

func (h *Handler) D() *mgo.Database {
	return h.DB.DB(h.Database)
}
