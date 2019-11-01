package handlers

import (
	"github.com/gocelery/gocelery"
	"github.com/gomodule/redigo/redis"
	"github.com/lab7arriam/cryweb/models"
	"github.com/lab7arriam/cryweb/providers"
	"github.com/labstack/echo/v4"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"time"
)

type Handler struct {
	DB       *mgo.Session
	Database string
	Key      string
	ES       *providers.EmailSender
	Url      string
	Route    func(name string, params ...interface{}) string
	WorkDir  string
	Celery   *gocelery.CeleryClient
	Threads  int
}

func (h *Handler) D() *mgo.Database {
	return h.DB.DB(h.Database)
}

func (h *Handler) DbUser() *mgo.Collection {
	return h.D().C("users")
}

func (h *Handler) DbTask() *mgo.Collection {
	return h.D().C("tasks")
}

func (h *Handler) InitCelery(redis_url string, w int) error {
	redisPool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(redis_url)
			if err != nil {
				return nil, err
			}
			return c, err
		},
	}

	cli, err := gocelery.NewCeleryClient(
		gocelery.NewRedisBroker(redisPool),
		&gocelery.RedisCeleryBackend{Pool: redisPool},
		w)
	if err != nil {
		return err
	}

	cry_processing := func(run_mode, fi, fr, rr, meta, wd string) (bool, string) {
		aresult, err := cli.Delay("cryprocess", run_mode, fi, fr, rr, meta, wd, h.Threads)
		if err != nil {
			return false, err.Error()
		}

		_, err = aresult.Get(time.Hour * 24 * 7) //TODO: extract timeout to config
		if err != nil {
			return false, err.Error()
		}

		var task *models.Task

		if err := h.DbTask().Find(bson.M{"work_dir": wd}).One(task); err != nil {
			return false, err.Error()
		}

		task.Finished = time.Now()
		task.Status = "finished"

		if err := h.DbTask().UpdateId(task.Id, task); err != nil {
			return false, err.Error()
		}

		if err := h.ES.Send([]string{task.UserId},
			"Task is completed",
			"complited", echo.Map{
				"tool": task.Tool,
				"url":  h.Route("tasks.result", task.Tool, task.Id.String()),
			}); err != nil {
			return false, err.Error()
		}

		return true, ""
	}

	cli.Register("go_cry", cry_processing)

	cli.StartWorker()

	h.Celery = cli

	return err
}
