package redisutil

import (
	"errors"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	log "github.com/z9905080/gloger"
)

// gRedisConnectionPool => key: host ,value: connectionPool
var (
	gRedisConnectionPool = make(map[string]*redis.Pool, 0)
	mapLock              = new(sync.RWMutex)
)

// getConnectionPool 取得連線池
func getConnectionPool(server string, password string) *redis.Pool {
	mapLock.Lock()
	defer mapLock.Unlock()

	// 檢查是否存在，如果存在且沒有nil的話就取出來使用
	if redisConnectionPool, isExist := gRedisConnectionPool[server]; isExist && redisConnectionPool != nil {
		return redisConnectionPool
	}

	// 如果沒有就組一個新的並拿出來用
	gRedisConnectionPool[server] = &redis.Pool{
		Wait:        true,
		MaxIdle:     100,
		MaxActive:   1500,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server, redis.DialPassword(password))
			if err != nil {
				log.Error(err.Error())
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			if err != nil {
				//log.Warn(err.Error())
			}
			return err
		},
	}

	return gRedisConnectionPool[server]

}

// NewConnect 建立連線
func NewConnect(redisHost string, password string) (*redis.Pool, error) {
	//redisHost := "192.168.1.133:6379"
	//pw := "pass.123"
	if redisHost == "" {
		// 一定連不到
		return &redis.Pool{}, errors.New("未找到指定的連線目標設定檔")
	}

	return getConnectionPool(redisHost, password), nil
}
