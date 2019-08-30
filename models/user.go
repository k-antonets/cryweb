package models

import "golang.org/x/crypto/bcrypt"

type User struct {
	Email            string   `json:"email" bson:"email"`
	Hash             []byte   `json:"hash" bson:"hash"`
	Organisation     string   `json:"organisation" bson:"organisation"`
	Country          string   `json:"country" bson:"country"`
	ActivatedByMail  bool     `json:"activated_by_mail" bson:"activated_by_mail"`
	ActivatedByAdmin bool     `json:"activated_by_admin" bson:"activated_by_admin"`
	HasRunning       bool     `json:"has_running" bson:"has_running,omitempty"`
	Tasks            []string `json:"tasks" bson:"tasks,omitempty"`
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
