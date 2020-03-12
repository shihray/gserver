// Other options for implementations of the interface
// can be stored in a context
// Context context.Context

package registry

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
)

type redisKey string

const RegistRedisKey redisKey = "gserver:modules"

// fmt Addr return String
func (key redisKey) Addr(val string) string {
	return fmt.Sprintf("%s:%s", key, val)
}

type Options struct {
	Addrs     []string
	Timeout   time.Duration
	Secure    bool
	TLSConfig *tls.Config
	Context   context.Context

	RedisConn redis.Conn
}

type RegisterOptions struct {
	TTL     time.Duration
	Context context.Context
}

// Specify a service to watch
// If blank, the watch is for all services
type WatchOptions struct {
	Service string
	Context context.Context
}

// Addrs is the registry addresses to use
func Addrs(addrs ...string) Option {
	return func(o *Options) {
		o.Addrs = addrs
	}
}

func Timeout(t time.Duration) Option {
	return func(o *Options) {
		o.Timeout = t
	}
}

// Secure communication with the registry
func Secure(b bool) Option {
	return func(o *Options) {
		o.Secure = b
	}
}

// Specify TLS Config
func TLSConfig(t *tls.Config) Option {
	return func(o *Options) {
		o.TLSConfig = t
	}
}

func RegisterTTL(t time.Duration) RegisterOption {
	return func(o *RegisterOptions) {
		o.TTL = t
	}
}

// Watch a service
func WatchService(name string) WatchOption {
	return func(o *WatchOptions) {
		o.Service = name
	}
}

func RedisConn(r redis.Conn) Option {
	return func(o *Options) {
		o.RedisConn = r
	}
}
