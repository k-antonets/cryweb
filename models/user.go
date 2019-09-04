package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Email            string    `json:"email" bson:"_id" form:"email" validate:"required,email"`
	Hash             []byte    `json:"hash" bson:"hash"`
	Organisation     string    `json:"organisation" bson:"organisation" form:"organisation" validate:"required"`
	Country          string    `json:"country" bson:"country" form:"country" validate:"required"`
	ActivatedByMail  bool      `json:"activated_by_mail" bson:"activated_by_mail"`
	ActivatedByAdmin bool      `json:"activated_by_admin" bson:"activated_by_admin"`
	HasRunning       bool      `json:"has_running" bson:"has_running,omitempty"`
	Tasks            []string  `json:"tasks" bson:"tasks,omitempty"`
	Role             string    `json:"role" bson:"role"`
	FirstName        string    `json:"first_name" bson:"first_name" form:"first_name" validate:"required"`
	LastName         string    `json:"last_name" bson:"last_name" form:"last_name" validate:"required"`
	Created          time.Time `json:"created" bson:"created"`
}

func (u *User) CheckPassword(pwd string) bool {
	bytePwd := []byte(pwd)

	err := bcrypt.CompareHashAndPassword(u.Hash, bytePwd)
	if err != nil {
		return false
	}

	return true
}

func (u *User) SetPassword(pwd string) error {
	bytePwd := []byte(pwd)
	hash, err := bcrypt.GenerateFromPassword(bytePwd, bcrypt.MinCost)

	if err != nil {
		return err
	}
	u.Hash = hash
	return nil
}

func (u *User) IsActive() bool {
	return u.ActivatedByMail && u.ActivatedByAdmin
}

func (u *User) CanRun() bool {
	return u.ActivatedByMail && u.ActivatedByAdmin && !u.HasRunning
}

func (u *User) AddTask(id string) {
	u.Tasks = append(u.Tasks, id)
}

func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

func NewUser() *User {
	return &User{
		Email:            "",
		Hash:             nil,
		Organisation:     "",
		Country:          "",
		ActivatedByMail:  false,
		ActivatedByAdmin: false,
		HasRunning:       false,
		Tasks:            []string{},
		Role:             "user",
		FirstName:        "",
		LastName:         "",
		Created:          time.Now(),
	}
}
