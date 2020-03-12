package basemodule

import (
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/shihray/gserver/registry"
	mqrpc "github.com/shihray/gserver/rpc"
	rpcpb "github.com/shihray/gserver/rpc/pb"
	"github.com/shihray/gserver/selector"
)

type Option func(*Options)

type Options struct {
	Nats             *nats.Conn
	Version          string
	Debug            bool
	WorkDir          string
	ConfPath         string
	LogDir           string
	BIDir            string
	ProcessID        string
	Registry         registry.Registry
	Selector         selector.Selector
	RegisterTTL      time.Duration
	RegisterInterval time.Duration
	ClientRPChandler ClientRPChandler
	ServerRPCHandler ServerRPCHandler
}

type ClientRPChandler func(app App, server registry.Service, rpcinfo rpcpb.RPCInfo, result interface{}, err string, exec_time int64)

type ServerRPCHandler func(app App, module Module, callInfo mqrpc.CallInfo)

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
