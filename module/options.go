package basemodule

import (
	"time"

	defaultRPC "github.com/shihray/gserver/rpc/base"

	NatsClient "github.com/nats-io/nats.go"
	"github.com/shihray/gserver/registry"
	mqRPC "github.com/shihray/gserver/rpc"
	rpcpb "github.com/shihray/gserver/rpc/pb"
)

type Option func(*Options)

type Options struct {
	Nats             *NatsClient.Conn
	NatsPool         *defaultRPC.Pool
	Version          string
	Debug            bool
	WorkDir          string
	ConfPath         string
	LogDir           string
	BIDir            string
	ProcessID        string
	Registry         registry.Registry
	RegisterTTL      time.Duration
	RegisterInterval time.Duration
	ClientRPChandler ClientRPChandler
	ServerRPCHandler ServerRPCHandler
}

type ClientRPChandler func(app App, server registry.Service, rpcinfo rpcpb.RPCInfo, result interface{}, err string, exec_time int64)

type ServerRPCHandler func(app App, module Module, callInfo mqRPC.CallInfo)

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

func Nats(nc *NatsClient.Conn) Option {
	return func(o *Options) {
		o.Nats = nc
	}
}

func NatsPool(np *defaultRPC.Pool) Option {
	return func(o *Options) {
		o.NatsPool = np
	}
}

// Registry sets the registry for the service
// and the underlying components
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

// RegisterInterval specifies the interval on which to re-register
func SetClientRPChandler(t ClientRPChandler) Option {
	return func(o *Options) {
		o.ClientRPChandler = t
	}
}

func SetServerRPCHandler(t ServerRPCHandler) Option {
	return func(o *Options) {
		o.ServerRPCHandler = t
	}
}
