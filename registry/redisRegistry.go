package registry

import (
	"fmt"
	hash "github.com/mitchellh/hashstructure"
	"strings"
	"sync"

	"github.com/gomodule/redigo/redis"
	Logging "github.com/shihray/gserver/logging"
	MyRedisUtil "github.com/shihray/gserver/source/redisutil"
)

type redisRegistry struct {
	sync.Mutex // Lock
	Address    string
	opts       Options
	connect    bool // connect enabled
	register   map[string]uint64
}

func newRedisRegistry(opts ...Option) Registry {
	cr := &redisRegistry{
		opts: Options{
			RedisHost:     "192.168.1.133:6379",
			RedisPassword: "pass.123",
		},
		register: make(map[string]uint64),
	}
	for _, o := range opts {
		o(&cr.opts)
	}

	//redisConn, connErr := cr.ConnectRedis()
	//if connErr != nil {
	//	return nil
	//}
	//defer redisConn.Close()
	//
	//keyList, err := redis.Strings(redisConn.Do("Keys", RegisterRedisKey.Addr("*")))
	//if err != nil && err != redis.ErrNil {
	//	msg := "redis KEYS Error, func: Register, 取得Redis資料Key錯誤 "
	//	Logging.Error(msg + err.Error())
	//	return nil
	//}
	//
	//for _, key := range keyList {
	//	redisConn.Do("DEL", key)
	//}

	return cr
}

func (c *redisRegistry) Deregister(s *Service) error {
	// delete the service
	redisConn, connErr := c.ConnectRedis()
	if connErr != nil {
		return connErr
	}
	defer redisConn.Close()

	redisName := fmt.Sprintf("%v@%v", s.Name, s.Address)

	_, err := redisConn.Do("HDEL", RegisterRedisKey.Addr(s.Name), redisName)
	if err != nil && err != redis.ErrNil {
		msg := "redis KEYS Error, func: Deregister, 取得Redis資料Key錯誤 "
		Logging.Error(msg + err.Error())
		return err
	}

	// delete our hash of the service
	c.Lock()
	delete(c.register, redisName)
	c.Unlock()

	return nil
}

func (c *redisRegistry) Register(s *Service, opts ...RegisterOption) error {
	// create hash of service; uint64
	h, err := hash.Hash(s, nil)
	if err != nil {
		return err
	}
	// get existing hash
	c.Lock()
	v, ok := c.register[s.Name]
	c.Unlock()

	// if it's already registered and matches then just pass the check
	if ok && v == h {
		return nil
	}
	// register the service
	redisConn, connErr := c.ConnectRedis()
	if connErr != nil {
		return connErr
	}
	defer redisConn.Close()

	redisName := fmt.Sprintf("%v@%v", s.Name, s.Address)
	if _, errOfRedis := redisConn.Do("HSET", RegisterRedisKey.Addr(s.Name), redisName, s.Address); errOfRedis != nil {
		msg := "redis KEYS Error, func: Register, 取得Redis資料Key錯誤 "
		Logging.Error(msg + errOfRedis.Error())
		return nil
	}
	// save our hash of the service
	c.Lock()
	c.register[redisName] = h
	c.Unlock()

	return nil
}

func (c *redisRegistry) GetService(name string) ([]*Service, error) {
	redisConn, connErr := c.ConnectRedis()
	if connErr != nil {
		return nil, connErr
	}
	defer redisConn.Close()

	var err error
	hList := make(map[string]string, 0)
	nameSplit := strings.Split(name, "@")
	if len(nameSplit) == 1 {
		hList, err = redis.StringMap(redisConn.Do("HGETALL", RegisterRedisKey.Addr(name)))
		if err != nil {
			msg := "redis KEYS Error, func: GetService, 取得Redis資料Key錯誤 "
			Logging.Error(msg + err.Error())
			return nil, err
		}
	} else {
		addr, err := redis.String(redisConn.Do("HGET", RegisterRedisKey.Addr(nameSplit[0]), name))
		if err != nil {
			msg := "redis KEYS Error, func: GetService, 取得Redis資料Key錯誤 "
			Logging.Error(msg + err.Error())
			return nil, err
		}
		hList[name] = addr
	}

	var services []*Service
	for redisName, address := range hList {
		svc := &Service{
			Name:    name,
			ID:      redisName,
			Address: address,
		}
		services = append(services, svc)
	}

	return services, nil
}

func (c *redisRegistry) ListServices() ([]*Service, error) {
	//rsp, _, err := c.Client.Catalog().Services(nil)
	//if err != nil {
	//	return nil, err
	//}

	var services []*Service

	//for service := range rsp {
	//	services = append(services, &Service{Name: service})
	//}

	return services, nil
}

func (c *redisRegistry) String() string {
	return "redisRegistry"
}

func (c *redisRegistry) Options() Options {
	return c.opts
}

func (c *redisRegistry) ConnectRedis() (redis.Conn, error) {
	redisPool, getPoolErr := MyRedisUtil.NewConnect(c.Options().RedisHost, c.Options().RedisPassword)
	if getPoolErr != nil {
		msg := "redis 連線錯誤,func: Register, 取得連線池失敗 "
		Logging.Error(msg + getPoolErr.Error())
		return nil, getPoolErr
	}
	return redisPool.Get(), getPoolErr
}
