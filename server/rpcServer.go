package server

import (
	"fmt"
	"sync"

	module "github.com/shihray/gserver/module"
	registry "github.com/shihray/gserver/registry"
	mqrpc "github.com/shihray/gserver/rpc"
	defaultrpc "github.com/shihray/gserver/rpc/base"
	conf "github.com/shihray/gserver/utils/conf"

	"github.com/shihray/gserver/logging"
	"github.com/shihray/gserver/utils/addr"
)

type rpcServer struct {
	exit chan chan error

	sync.RWMutex
	opts       Options
	server     mqrpc.RPCServer
	id         string
	registered bool           // used for first registration
	wg         sync.WaitGroup // graceful exit
}

func newRpcServer(opts ...Option) Server {
	options := newOptions(opts...)
	return &rpcServer{
		opts: options,
		exit: make(chan chan error),
	}
}

func (s *rpcServer) Options() Options {
	s.RLock()
	opts := s.opts
	s.RUnlock()
	return opts
}

func (s *rpcServer) Init(opts ...Option) error {
	s.Lock()
	for _, opt := range opts {
		opt(&s.opts)
	}
	s.Unlock()
	return nil
}

func (s *rpcServer) OnInit(module module.Module, app module.App, settings *conf.ModuleSettings) error {
	server, err := defaultrpc.NewRPCServer(app, module) //默認會創建一個本地的RPC
	if err != nil {
		logging.Warn("Dial: %s", err)
	}
	s.server = server
	s.opts.Address = server.Addr()
	if err := s.ServiceRegister(); err != nil {
		return err
	}
	return nil
}

func (s *rpcServer) SetListener(listener mqrpc.RPCListener) {
	s.server.SetListener(listener)
}

func (s *rpcServer) Register(id string, f interface{}) {
	if s.server == nil {
		panic("invalid RPCServer")
	}
	s.server.Register(id, f)
}

func (s *rpcServer) RegisterGO(id string, f interface{}) {
	if s.server == nil {
		panic("invalid RPCServer")
	}
	s.server.RegisterGO(id, f)
}

func (s *rpcServer) ServiceRegister() error {
	// parse address for host, port
	config := s.Options()
	var confAddr string = config.Address

	addr, err := addr.Extract(confAddr)
	if err != nil {
		return err
	}

	// register service
	service := &registry.Service{
		Name:     config.Name,
		ID:       config.Name + "@" + config.ID,
		Address:  addr,
		Metadata: config.Metadata,
	}
	s.id = service.ID
	service.Metadata["server"] = s.String()
	service.Metadata["registry"] = config.Registry.String()

	s.Lock()
	registered := s.registered
	s.Unlock()

	if !registered {
		logging.Info("Registering node: %s", service.ID)
	}

	// create registry options
	rOpts := []registry.RegisterOption{
		registry.RegisterTTL(config.RegisterTTL),
	}

	if err := config.Registry.Register(service, rOpts...); err != nil {
		return err
	}

	// already registered? don't need to register subscribers
	if registered {
		return nil
	}

	s.Lock()
	defer s.Unlock()

	s.registered = true

	return nil
}

func (s *rpcServer) ServiceDeregister() error {
	config := s.Options()
	var confAddr string = config.Address

	addr, err := addr.Extract(confAddr)
	if err != nil {
		return err
	}
	fmt.Printf("addr: %s\n", addr)

	// register service
	service := &registry.Service{
		Name:    config.Name,
		ID:      config.Name + "@" + config.ID,
		Address: addr,
	}

	logging.Info("Deregistering node: %s", service.ID)
	if err := config.Registry.Deregister(service); err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()
	if !s.registered {
		return nil
	}
	s.registered = false
	return nil
}

func (s *rpcServer) Start() error {
	return nil
}

func (s *rpcServer) Stop() error {
	if s.server != nil {
		logging.Info("RPCServer closeing id(%s)", s.id)
		err := s.server.Done()
		if err != nil {
			logging.Warn("RPCServer close fail id(%s) error(%s)", s.id, err)
		} else {
			logging.Info("RPCServer close success id(%s)", s.id)
		}
		s.server = nil
	}
	return nil
}

func (s *rpcServer) OnDestroy() error {
	return s.Stop()
}

func (s *rpcServer) ID() string {
	return s.id
}

func (s *rpcServer) String() string {
	return "rpc"
}
