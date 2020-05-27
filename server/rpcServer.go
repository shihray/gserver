package server

import (
	"fmt"
	"sync"

	module "github.com/shihray/gserver/module"
	ModuleRegistry "github.com/shihray/gserver/registry"
	mqRPC "github.com/shihray/gserver/rpc"
	defaultRPC "github.com/shihray/gserver/rpc/base"
	CommonConf "github.com/shihray/gserver/utils/conf"

	"github.com/shihray/gserver/utils/addr"
	log "github.com/z9905080/gloger"
)

type rpcServer struct {
	exit chan chan error

	*sync.RWMutex
	opts       Options
	server     mqRPC.RPCServer
	id         string
	registered bool           // used for first registration
	wg         sync.WaitGroup // graceful exit
}

func newRpcServer(opts ...Option) Server {
	options := newOptions(opts...)
	return &rpcServer{
		RWMutex: new(sync.RWMutex),
		opts:    options,
		exit:    make(chan chan error),
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

func (s *rpcServer) OnInit(module module.Module, app module.App, settings *CommonConf.ModuleSettings) error {
	server, err := defaultRPC.NewRPCServer(app, module) //默認會創建一個本地的RPC
	if err != nil {
		log.Warn("Dial Error:", err)
	}
	s.server = server
	s.opts.Address = server.Addr()
	if err := s.ServiceRegister(); err != nil {
		return err
	}
	return nil
}

func (s *rpcServer) SetListener(listener mqRPC.RPCListener) {
	s.server.SetListener(listener)
}

func (s *rpcServer) SetGoroutineControl(control mqRPC.GoroutineControl) {
	s.server.SetGoroutineControl(control)
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
	service := &ModuleRegistry.Service{
		Name:     config.Name,
		ID:       config.Name + "@" + config.ID,
		Address:  addr,
		Metadata: config.Metadata,
	}
	s.id = service.ID
	service.Metadata["server"] = s.String()
	service.Metadata["ModuleRegistry"] = config.Registry.String()

	s.Lock()
	registered := s.registered
	s.Unlock()

	if !registered {
		log.Info("Registering node:", service.ID)
	}

	config.Registry.Clean(service.Name)
	//config.Registry.ListServices()
	// create ModuleRegistry options
	rOpts := []ModuleRegistry.RegisterOption{}
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
	service := &ModuleRegistry.Service{
		Name:    config.Name,
		ID:      config.Name + "@" + config.ID,
		Address: addr,
	}

	log.Info("Deregistering node: ", service.ID)
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
		log.Info("RPCServer closeing id(%s)", s.id)
		err := s.server.Done()
		if err != nil {
			log.Warn("RPCServer close fail id(%s) error(%s)", s.id, err)
		} else {
			log.Info("RPCServer close success id(%s)", s.id)
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
