package registry

import (
	"crypto/tls"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	hash "github.com/mitchellh/hashstructure"

	"github.com/gomodule/redigo/redis"
	consul "github.com/hashicorp/consul/api"
	Logging "github.com/shihray/gserver/logging"
	RedisUtil "github.com/shihray/gserver/source/redisutil"
)

type redisRegistry struct {
	sync.Mutex
	Address  string
	Client   *consul.Client
	opts     Options
	connect  bool // connect enabled
	register map[string]uint64
}

func getDeregisterTTL(t time.Duration) time.Duration {
	// splay slightly for the watcher?
	splay := time.Second * 5
	deregTTL := t + splay

	// consul has a minimum timeout on deregistration of 1 minute.
	if t < time.Minute {
		deregTTL = time.Minute + splay
	}

	return deregTTL
}

func newTransport(config *tls.Config) *http.Transport {
	if config == nil {
		config = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     config,
	}
	runtime.SetFinalizer(&t, func(tr **http.Transport) {
		(*tr).CloseIdleConnections()
	})
	return t
}

func newRedisRegistry(opts ...Option) Registry {
	cr := &redisRegistry{
		opts:     Options{},
		register: make(map[string]uint64),
	}
	return cr
}

func (c *redisRegistry) Deregister(s *Service) error {
	// delete the service
	redisConn, connErr := RedisUtil.AddConnect("RedisRegistry")
	if connErr != nil {
		msg := "redis 連線錯誤,func: Deregister, 取得連線失敗 "
		Logging.Error(msg + connErr.Error())
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
	// 發送註冊消息至Redis
	redisConn, connErr := RedisUtil.AddConnect("RedisRegistry")
	if connErr != nil {
		msg := "redis 連線錯誤,func: Register, 取得連線失敗 "
		Logging.Error(msg + connErr.Error())
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
	data, err := redis.Strings(c.opts.RedisConn.Do("SMEMBERS", RegistRedisKey.Addr(name)))
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
	rsp, _, err := c.Client.Catalog().Services(nil)
	if err != nil {
		return nil, err
	}

	var services []*Service

	for service := range rsp {
		services = append(services, &Service{Name: service})
	}

	return services, nil
}

func (c *redisRegistry) String() string {
	return "redisRegistry"
}

func (c *redisRegistry) Options() Options {
	return c.opts
}
