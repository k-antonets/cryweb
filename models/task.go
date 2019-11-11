package models

import (
	"io/ioutil"
	"time"

	"gopkg.in/mgo.v2/bson"
)

type Task struct {
	Id        bson.ObjectId     `json:"id" bson:"_id"`
	Name      string            `json:"name" bson:"name,omitempty"`
	Created   time.Time         `json:"created" bson:"created"`
	Status    string            `json:"status" bson:"status"`
	Finished  time.Time         `json:"finished" bson:"finished,omitempty"`
	Params    map[string]string `json:"params" bson:"params"`
	WorkDir   string            `json:"work_dir" bson:"work_dir"`
	Removed   bool              `json:"removed" bson:"removed"`
	UserId    string            `json:"user_id" bson:"user_id"`
	Tool      string            `json:"tool" bson:"tool"`
	TaskId    string            `json:"task_id" bson:"task_id"`
	Timeouted bool              `json:"timeouted" bson:"timeouted"`
}

func NewTask(user, tool, tmpDir string) (*Task, error) {
	work_dir, err := ioutil.TempDir(tmpDir, tool)
	if err != nil {
		return nil, err
	}
	return &Task{
		Id:      bson.NewObjectId(),
		Created: time.Now(),
		Status:  "created",
		WorkDir: work_dir,
		Removed: false,
		UserId:  user,
		Tool:    tool,
		Params:  make(map[string]string),
	}, nil
}

func (t *Task) ResultExists() bool {
	if t.Status != "finished" || t.Removed {
		return false
	}
	return true
}

func (t *Task) NotFinished() bool {
	return t.Status != "finished" && t.Status != "failed" && !t.Removed
}

func (t *Task) Finish() {
	t.Finished = time.Now()
	t.Status = "finished"
}

func (t *Task) Fail() {
	t.Finished = time.Now()
	t.Status = "failed"
}

func (t *Task) TimeoutFail() {
	t.Finished = time.Now()
	t.Status = "failed"
	t.Timeouted = true
}

func (t *Task) Failed() bool {
	return t.Status == "failed"
}

func (t *Task) Run(task_id string) {
	t.Status = "running"
	t.TaskId = task_id
}

func (t *Task) IsRunning() bool {
	return t.Status == "running"
}

func (t *Task) ResultAvailable(user string) bool {
	if user != t.UserId || !t.ResultExists() {
		return false
	}
	return true
}

func (t *Task) AddParam(name, value string) {
	t.Params[name] = value
}

func (t *Task) GetParam(name string) string {
	if result, ok := t.Params[name]; ok {
		return result
	} else {
		return ""
	}
}
