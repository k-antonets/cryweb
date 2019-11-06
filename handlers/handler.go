package handlers

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/k-antonets/gocelery"
	"github.com/lab7arriam/cryweb/models"
	"github.com/lab7arriam/cryweb/providers"
	"github.com/labstack/echo/v4"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"net/url"
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

func (h *Handler) InitCelery(redis_url string, w, timeout int) error {
	redisPool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(redis_url)
			if err != nil {
				return nil, err
			}
			return c, err
		},
	}

	fmt.Printf("number of workers is equal to %d\n", w)
	cli, err := gocelery.NewCeleryClient(
		gocelery.NewRedisBroker(redisPool, "cry_go"),
		&gocelery.RedisCeleryBackend{Pool: redisPool},
		w)
	if err != nil {
		return err
	}

	cry_processing := func(run_mode, fi, fr, rr, meta, wd string) (bool, string) {
		aresult, err := cli.DelayToQueue("tasks.cryprocess", "cry_py", run_mode, fi, fr, rr, meta, wd, h.Threads)
		if err != nil {
			return false, err.Error()
		}

		task := &models.Task{}

		if err := h.DbTask().Find(bson.M{"work_dir": wd}).One(task); err != nil {
			fmt.Printf("failed to get task for work_dir %s, error: %v\n", wd, err)
			return false, err.Error()
		}

		task.Status = "running"

		if err := h.DbTask().UpdateId(task.Id, task); err != nil {
			fmt.Printf("failed to update task %s entity, error: %v\n", task.Id.Hex(), err)
			return false, err.Error()
		}

		_, err = aresult.Get(time.Hour * time.Duration(timeout))
		if err != nil {
			return false, err.Error()
		}

		task.Finished = time.Now()
		task.Status = "finished"

		if err := h.DbTask().UpdateId(task.Id, task); err != nil {
			fmt.Printf("failed to update task %s entity, error: %v\n", task.Id.Hex(), err)
			return false, err.Error()
		}

		relative_url := h.Route("tasks.result", task.Tool, task.Id.Hex())
		absolute_url, err := url.Parse(h.Url)
		if err != nil {
			fmt.Printf("failed to parse server domain %s, error: %v\n", h.Url, err)
			return false, err.Error()
		}
		absolute_url.Path = relative_url

		if err := h.ES.Send([]string{task.UserId},
			"Task is completed",
			"completed", echo.Map{
				"tool": task.Tool,
				"url":  absolute_url.String(),
			}); err != nil {
			return false, err.Error()
		}

		return true, ""
	}

	fmt.Println("registering new task go_cry")
	cli.Register("go_cry", cry_processing)

	fmt.Println("starting workers")
	cli.StartWorker()

	h.Celery = cli

	return err
}
