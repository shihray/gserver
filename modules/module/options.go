package basemodule

import (
	nats "github.com/nats-io/nats.go"
)

type Option func(*Options)

type Options struct {
	Nats      *nats.Conn
	Version   string
	Debug     bool
	WorkDir   string
	ConfPath  string
	LogDir    string
	BIDir     string
	ProcessID string
}

func Version(v string) Option {
	return func(o *Options) {
		o.Version = v
	}
}

func WorkDir(v string) Option {
	return func(o *Options) {
		o.WorkDir = v
	}
}

func Configure(v string) Option {
	return func(o *Options) {
		o.ConfPath = v
	}
}

func LogDir(v string) Option {
	return func(o *Options) {
		o.LogDir = v
	}
}

func ProcessID(v string) Option {
	return func(o *Options) {
		o.ProcessID = v
	}
}

func Nats(nc *nats.Conn) Option {
	return func(o *Options) {
		o.Nats = nc
	}
}
