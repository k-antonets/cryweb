package handlers

import "gopkg.in/mgo.v2"

type Handler struct {
	DB       *mgo.Session
	Database string
	Key      string
}
