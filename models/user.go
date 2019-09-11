package models

import (
	"bytes"
	"net/url"
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

func (u *User) getActivationMsg(secret, admin string) []byte {
	msg := &bytes.Buffer{}

	msg.WriteString(u.Email)
	msg.WriteString(u.FirstName)
	msg.WriteString(u.LastName)
	msg.WriteString(u.Organisation)
	msg.WriteString(u.Created.Format(time.UnixDate))
	msg.WriteString(secret)
	msg.WriteString(admin)

	return msg.Bytes()
}

func (u *User) GetActivationUrl(secret, server_url string) (string, error) {

	hash, err := bcrypt.GenerateFromPassword(u.getActivationMsg(secret, ""), bcrypt.MinCost)

	if err != nil {
		return "", err
	}

	q := url.Values{}
	q.Add("email", u.Email)
	q.Add("hash", string(hash))

	ur, err := url.Parse(server_url)
	if err != nil {
		return "", err
	}
	ur.Path = "activate"
	ur.RawQuery = q.Encode()

	return ur.String(), nil
}

func (u *User) GetAdminActivationUrl(secret, server_url, admin string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(u.getActivationMsg(secret, admin), bcrypt.MinCost)

	if err != nil {
		return "", err
	}

	q := url.Values{}
	q.Add("email", u.Email)
	q.Add("hash", string(hash))
	q.Add("admin", admin)

	ur, err := url.Parse(server_url)
	if err != nil {
		return "", err
	}
	ur.Path = "activate"
	ur.RawQuery = q.Encode()

	return ur.String(), nil

}

func (u *User) ActivateEmail(secret, hash string) bool {
	if err := bcrypt.CompareHashAndPassword(u.getActivationMsg(secret, ""), []byte(hash)); err != nil {
		return false
	}
	u.ActivatedByMail = true
	return true
}

func (u *User) ActivateAdmin(secret, hash, admin string) bool {
	if err := bcrypt.CompareHashAndPassword(u.getActivationMsg(secret, admin), []byte(hash)); err != nil {
		return false
	}
	u.ActivatedByAdmin = true
	return true
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
