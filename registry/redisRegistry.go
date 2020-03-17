package registry

import (
	hash "github.com/mitchellh/hashstructure"
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

	redisConn, connErr := cr.ConnectRedis()
	if connErr != nil {
		return nil
	}
	defer redisConn.Close()

	keyList, err := redis.Strings(redisConn.Do("Keys", RegistRedisKey.Addr("*")))
	if err != nil && err != redis.ErrNil {
		msg := "redis KEYS Error, func: Register, 取得Redis資料Key錯誤 "
		Logging.Error(msg + err.Error())
		return nil
	}

	for _, key := range keyList {
		redisConn.Do("DEL", key)
	}

	return cr
}

func (c *redisRegistry) Deregister(s *Service) error {
	// delete the service
	redisConn, connErr := c.ConnectRedis()
	if connErr != nil {
		return connErr
	}
	defer redisConn.Close()

	_, err := redisConn.Do("SREM", RegistRedisKey.Addr(s.Name), s.Address)
	if err != nil && err != redis.ErrNil {
		msg := "redis KEYS Error, func: Deregister, 取得Redis資料Key錯誤 "
		Logging.Error(msg + err.Error())
		return err
	}

	// delete our hash of the service
	c.Lock()
	delete(c.register, s.Name)
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

	_, err = redisConn.Do("SADD", RegistRedisKey.Addr(s.Name), s.Address)
	if err != nil && err != redis.ErrNil {
		msg := "redis KEYS Error, func: Register, 取得Redis資料Key錯誤 "
		Logging.Error(msg + err.Error())
		return err
	}

	// save our hash of the service
	c.Lock()
	c.register[s.Name] = h
	c.Unlock()

	return nil
}

func (c *redisRegistry) GetService(name string) ([]*Service, error) {
	redisConn, connErr := c.ConnectRedis()
	if connErr != nil {
		return nil, connErr
	}
	defer redisConn.Close()

	data, err := redis.Strings(redisConn.Do("SMEMBERS", RegistRedisKey.Addr(name)))
	if err != nil && err != redis.ErrNil {
		msg := "redis KEYS Error, func: Register, 取得Redis資料Key錯誤 "
		Logging.Error(msg + err.Error())
		return nil, err
	}

	var services []*Service
	for _, address := range data {
		svc := &Service{
			Name:    name,
			ID:      name,
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
