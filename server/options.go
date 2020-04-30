package server

import (
	"context"
	"github.com/shihray/gserver/registry"
)

type Options struct {
	Registry registry.Registry

	Metadata map[string]string
	Name     string
	Address  string
	ID       string
	Version  string
	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context

	RoutineCount int // RPC Server Routine Channel Buffer Length
}

func newOptions(opt ...Option) Options {
	opts := Options{
		Metadata: map[string]string{},
	}

	for _, o := range opt {
		o(&opts)
	}

	if len(opts.Address) == 0 {
		opts.Address = DefaultAddress
	}

	if len(opts.Name) == 0 {
		opts.Name = DefaultName
	}

	if len(opts.ID) == 0 {
		opts.ID = DefaultID
	}

	if len(opts.Version) == 0 {
		opts.Version = DefaultVersion
	}

	if opts.Registry == nil {
		opts.Registry = registry.DefaultRegistry
	}

	if opts.RoutineCount == 0 {
		opts.RoutineCount = 100
	}

	return opts
}

// Server name
func Name(n string) Option {
	return func(o *Options) {
		o.Name = n
	}
}

// Unique server id
func ID(id string) Option {
	return func(o *Options) {
		o.ID = id
	}
}

// Version of the service
func Version(v string) Option {
	return func(o *Options) {
		o.Version = v
	}
}

// Address to bind to - host:port
func Address(a string) Option {
	return func(o *Options) {
		o.Address = a
	}
}

// Metadata associated with the server
func Metadata(md map[string]string) Option {
	return func(o *Options) {
		o.Metadata = md
	}
}

// Registry used for discovery
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

// Wait tells the server to wait for requests to finish before exiting
func Wait(b bool) Option {
	return func(o *Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, "wait", b)
	}
}

func RoutineCount(num int) Option {
	return func(o *Options) {
		o.RoutineCount = num
	}
}
