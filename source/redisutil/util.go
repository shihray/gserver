package redisutil

import (
	"errors"
	"sync"
	"time"

	Logging "github.com/shihray/gserver/logging"

	"github.com/BurntSushi/toml"
	"github.com/garyburd/redigo/redis"
)

// RedisConf 設定檔全域變數
var RedisConf Config

// DetailSet 設定檔內容
type DetailSet struct {
	IP       string `toml:"ip"`
	Port     string `toml:"port"`
	Password string `toml:"password"`
}

// PublishSet 設定檔內容
type PublishSet struct {
	IP      string `toml:"ip"`
	Port    string `toml:"port"`
	Channel string `toml:"channel"`
}

// GetHostString 取得Host連線字串
func (ds DetailSet) GetHostString() string {
	return ds.IP + ":" + ds.Port
}

// GetHostString 取得Host連線字串
func (ps PublishSet) GetHostString() string {
	return ps.IP + ":" + ps.Port
}

// Config 設定檔
type Config struct {
	Redis struct {
		GameRedisMaster DetailSet `toml:"game_redis_master"`
		GameRedisSlave  DetailSet `toml:"game_redis_slave"`
		BackendMaster   DetailSet `toml:"backend_master"`
		BackendSlave    DetailSet `toml:"backend_slave"`
	} `toml:"redis"`
	MaxLink struct {
		Max int `toml:"max"`
	} `toml:"maxlink"`
}

// gRedisConnectionPool => key: host ,value: connectionPool
var gRedisConnectionPool = make(map[string]*redis.Pool, 0)

var mapLock sync.Mutex

// Init 初始化
func Init(path string) error {
	if _, err := toml.DecodeFile(path, &RedisConf); err != nil {
		msg := "Init, 解析Redis設定檔發生錯誤 "
		Logging.Fatal(msg + err.Error())
		return errors.New(msg)
	}
	return nil
}

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
				Logging.Error(err.Error())
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			if err != nil {
				//Logging.Warn(err.Error())
			}
			return err
		},
	}

	return gRedisConnectionPool[server]

}

// AddConnect 建立連線
func AddConnect(redisName string) (redis.Conn, error) {
	var (
		redisHost string
		pw        string
	)
	switch redisName {
	case "GameRedisMaster":
		redisHost = RedisConf.Redis.GameRedisMaster.GetHostString()
		pw = RedisConf.Redis.GameRedisMaster.Password
	case "GameRedisSlave":
		redisHost = RedisConf.Redis.GameRedisSlave.GetHostString()
		pw = RedisConf.Redis.GameRedisSlave.Password
	case "BackendMaster":
		redisHost = RedisConf.Redis.BackendMaster.GetHostString()
		pw = RedisConf.Redis.BackendMaster.Password
	case "BackendSlave":
		redisHost = RedisConf.Redis.BackendSlave.GetHostString()
		pw = RedisConf.Redis.BackendSlave.Password

	}

	if redisHost == "" {
		// 一定連不到
		c, _ := redis.Dial("tcp", "")
		return c, errors.New("未找到指定的連線目標設定檔")
	}

	redisConnPool := getConnectionPool(redisHost, pw)

	redisConn := redisConnPool.Get()
	return redisConn, nil
}
