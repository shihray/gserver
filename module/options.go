package basemodule

import (
	CommonNats "github.com/nats-io/nats.go"
	"github.com/shihray/gserver/registry"
	mqrpc "github.com/shihray/gserver/rpc"
	rpcPB "github.com/shihray/gserver/rpc/pb"
	"time"
)

type Option func(*Options)

type Options struct {
	Nats             *CommonNats.Conn
	Version          string
	ConfPath         string
	LogMode          int
	LogLevel         int
	BIDir            string
	Registry         registry.Registry
	ClientRPChandler ClientRPChandler
	ServerRPCHandler ServerRPCHandler
	RoutineCount     int
	// Register loop interval
	RegisterInterval time.Duration
	ExitSignal       chan bool
}

type ClientRPChandler func(app App, server registry.Service, rpcinfo rpcPB.RPCInfo, result interface{}, err string, exec_time int64)

type ServerRPCHandler func(app App, module Module, callInfo mqrpc.CallInfo)

func Version(v string) Option {
	return func(o *Options) {
		o.Version = v
	}
}

func LogMode(v int) Option {
	return func(o *Options) {
		o.LogMode = v
	}
}

func LogLevel(v int) Option {
	return func(o *Options) {
		o.LogLevel = v
	}
}

func Configure(v string) Option {
	return func(o *Options) {
		o.ConfPath = v
	}
}

func Nats(nc *CommonNats.Conn) Option {
	return func(o *Options) {
		o.Nats = nc
	}
}

// Registry sets the registry for the service
// and the underlying components
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

func RegisterInterval(t time.Duration) Option {
	return func(o *Options) {
		o.RegisterInterval = t
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

func RoutineCount(num int) Option {
	return func(o *Options) {
		o.RoutineCount = num
	}
}

func ExitSignal(ok chan bool) Option {
	return func(o *Options) {
		o.ExitSignal = ok
	}
}
