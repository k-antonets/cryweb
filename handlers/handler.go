package handlers

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/k-antonets/gocelery"
	"github.com/lab7arriam/cryweb/models"
	"github.com/lab7arriam/cryweb/providers"
	"github.com/labstack/echo/v4"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
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

func (h *Handler) InitCelery(redis_url string, w, timeout int, support string) error {
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

		task := &models.Task{}

		if err := h.DbTask().Find(bson.M{"work_dir": wd}).One(task); err != nil {
			fmt.Printf("failed to get task for work_dir %s, error: %v\n", wd, err)
			return false, err.Error()
		}

		aresult := cli.GetAsyncResult(task.TaskId)
		err := errors.New("")

		fmt.Printf("beginning task %s with celery id <%s>\n", task.Id.Hex(), task.TaskId)
		if task.TaskId == "" {

			aresult, err = cli.DelayToQueue("tasks.cryprocess", "cry_py", run_mode, fi, fr, rr, meta, wd, h.Threads)
			if err != nil {
				return false, err.Error()
			}

			task.Run(aresult.TaskID)

			if err := h.DbTask().UpdateId(task.Id, task); err != nil {
				fmt.Printf("failed to update task %s entity, error: %v\n", task.Id.Hex(), err)
				return false, err.Error()
			}
		}

		_, err = aresult.Get(time.Minute * time.Duration(timeout))
		if err != nil {
			fmt.Printf("failed to get results for task %s, error: %s\n", task.Id.Hex(), err.Error())
			switch err.(type) {
			case *gocelery.ErrTaskFailure:
				task.Fail()
			default:
				task.TimeoutFail()
			}
		} else {
			task.Finish()
		}

		if err := h.DbTask().UpdateId(task.Id, task); err != nil {
			fmt.Printf("failed to update task %s entity, error: %v\n", task.Id.Hex(), err)
			return false, err.Error()
		}

		if task.Failed() {
			if err := h.ES.Send([]string{task.UserId},
				"Task is failed",
				"failed", echo.Map{
					"timeout": task.Timeouted,
					"label":   task.Name,
					"tool":    task.Tool,
					"support": support,
				}); err != nil {
				return false, err.Error()
			}
			return false, fmt.Sprintf("task %s has been failed", task.Id.Hex())
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
				"label": task.Name,
				"tool":  task.Tool,
				"url":   absolute_url.String(),
			}); err != nil {
			return false, err.Error()
		}

		return true, ""
	}

	fmt.Println("registering new task go_cry")
	cli.Register("go_cry", cry_processing)

	fmt.Println("starting workers")
	cli.StartWorker()

	//Run all unfinished tasks after restart
	unfinished_tasks := []*models.Task{}
	if err := h.DbTask().Find(bson.M{"removed": false, "status": bson.M{"$in": []string{"created", "running"}}}).All(&unfinished_tasks); err != nil {
		fmt.Printf("failed to get unfinished tasks, error: %v\n", err)
		return err
	}
	fmt.Printf("left unfinished tasks: %d\n", len(unfinished_tasks))
	for _, task := range unfinished_tasks {
		fmt.Printf("restarting task %s with celery id <%s>\n", task.Id.Hex(), task.TaskId)
		if _, err := cli.Delay("go_cry", task.GetParam("run_mode"), task.GetParam("fi"),
			task.GetParam("fo"), task.GetParam("re"), task.GetParam("meta"),
			task.WorkDir); err != nil {
			return err
		}
	}

	h.Celery = cli

	return err
}
