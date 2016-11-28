package workers

import (
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
)

type scheduled struct {
	keys   []string
	closed chan bool
	exit   chan bool
}

func (s *scheduled) start() {
	go (func() {
		for {
			s.poll()

			select {
			case <-s.closed:
				return
			case <-time.After(time.Duration(Config.PollInterval) * time.Second):
			}

		}
	})()
}

func (s *scheduled) quit() {
	close(s.closed)
}

func (s *scheduled) poll() {
	conn := Config.Pool.Get()

	now := nowToSecondsWithNanoPrecision()

	for _, key := range s.keys {
		key = Config.Namespace + key
		for {
			messages, _ := redis.Strings(conn.Do("zrangebyscore", key, "-inf", now, "limit", 0, 1))

			if len(messages) == 0 {
				break
			}

			message, _ := NewMsgFromString(messages[0])

			if removed, _ := redis.Bool(conn.Do("zrem", key, messages[0])); removed {
				queue := strings.TrimPrefix(message.queue, Config.Namespace)
				message.enqueuedAt = nowToSecondsWithNanoPrecision()
				conn.Do("lpush", Config.Namespace+"queue:"+queue, message.ToJson())
			}
		}
	}

	conn.Close()
}

func newScheduled(keys ...string) *scheduled {
	return &scheduled{keys, make(chan bool), make(chan bool)}
}
