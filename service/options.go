package service

import (
	"context"
	"time"

	"github.com/shihray/gserver/server"
)

type Service interface {
	Init(...Option)
	Options() Options
	Server() server.Server
	Run() error
	String() string
}

type Option func(*Options)

type Options struct {
	Server server.Server

	// Register loop interval
	RegisterInterval time.Duration

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

func newOptions(opts ...Option) Options {
	opt := Options{
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

// Context specifies a context for the service.
// Can be used to signal shutdown of the service.
// Can be used for extra option values.
func Context(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}

func Server(s server.Server) Option {
	return func(o *Options) {
		o.Server = s
	}
}

func RegisterInterval(t time.Duration) Option {
	return func(o *Options) {
		o.RegisterInterval = t
	}
}
