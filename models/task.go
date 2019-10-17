package models

import (
	"io/ioutil"
	"time"

	"gopkg.in/mgo.v2/bson"
)

type Task struct {
	Id       bson.ObjectId     `json:"id" bson:"_id"`
	Name     string            `json:"name" bson:"name,omitempty"`
	Created  time.Time         `json:"created" bson:"created"`
	Status   string            `json:"status" bson:"status"`
	Finished time.Time         `json:"finished" bson:"finished,omitempty"`
	Params   map[string]string `json:"params" bson:"params"`
	WorkDir  string            `json:"work_dir" bson:"work_dir"`
	Removed  bool              `json:"removed" bson:"removed"`
	UserId   string            `json:"user_id" bson:"user_id"`
	Tool     string            `json:"tool" bson:"tool"`
}

func NewTask(user, tool, tmpDir string) (*Task, error) {
	work_dir, err := ioutil.TempDir(tmpDir, tool)
	if err != nil {
		return nil, err
	}
	return &Task{
		Created: time.Now(),
		Status:  "created",
		WorkDir: work_dir,
		Removed: false,
		UserId:  user,
		Tool:    tool,
	}, nil
}

func (t *Task) ResultAvailable(user string) bool {
	if user != t.UserId || t.Status != "finished" || t.Removed {
		return false
	}
	return true
}