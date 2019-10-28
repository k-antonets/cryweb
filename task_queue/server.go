package task_queue

import (
	"github.com/gocelery/gocelery"
	"github.com/gomodule/redigo/redis"
)

func Init(redis_url string, w int) {
	redisPool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(redis_url)
			if err != nil {
				return nil, err
			}
			return c, err
		},
	}

	cli, _ := gocelery.NewCeleryClient(
		gocelery.NewRedisBroker(redisPool),
		&gocelery.RedisCeleryBackend{Pool: redisPool},
		w)

	cli.StartWorker()
}
