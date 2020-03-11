package basemodule

import (
	logging "github.com/shihray/gserver/modules/logging"
	module "github.com/shihray/gserver/modules/module"
	mqrpc "github.com/shihray/gserver/modules/rpc"
	"github.com/shihray/gserver/modules/utils/conf"
)

type rpcserver struct {
	settings *conf.ModuleSettings
	server   mqrpc.RPCServer
}

func (s *rpcserver) GetId() string {
	return s.settings.ID
}

func (s *rpcserver) OnInit(module module.Module, app module.App, settings *conf.ModuleSettings) {
	s.settings = settings
	server, err := defaultrpc.NewRPCServer(app, module) // 默认会创建一个本地的RPC
	if err != nil {
		logging.Warning("Dial: %s", err)
	}

	s.server = server
	logging.Info("RPCServer init success id(%s) version(%s)", s.settings.ID, module.Version())
}

func (s *rpcserver) OnDestroy() {
	if s.server != nil {
		logging.Info("RPCServer closeing id(%s)", s.settings.ID)
		err := s.server.Done()
		if err != nil {
			logging.Warning("RPCServer close fail id(%s) error(%s)", s.settings.ID, err)
		} else {
			logging.Info("RPCServer close success id(%s)", s.settings.ID)
		}
		s.server = nil
	}
}

func (s *rpcserver) Register(id string, f interface{}) {
	if s.server == nil {
		panic("invalid RPCServer")
	}
	s.server.Register(id, f)
}

func (s *rpcserver) RegisterGO(id string, f interface{}) {
	if s.server == nil {
		panic("invalid RPCServer")
	}
	s.server.RegisterGO(id, f)
}

func (s *rpcserver) GetRPCServer() mqrpc.RPCServer {
	if s.server == nil {
		panic("invalid RPCServer")
	}
	return s.server
}
