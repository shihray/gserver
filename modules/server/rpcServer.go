package server

import (
	"strconv"
	"strings"
	"sync"

	module "github.com/shihray/gserver/modules/module"
	defaultrpc "github.com/shihray/gserver/modules/rpc/base"
	conf "github.com/shihray/gserver/modules/utils/conf"

	"github.com/liangdas/mqant/registry"
	"github.com/shihray/gserver/modules/logging"
	"github.com/shihray/gserver/modules/utils/addr"
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
	// update internal server

	s.Unlock()
	return nil
}

func (s *rpcServer) OnInit(module module.Module, app module.App, settings *conf.ModuleSettings) error {
	server, err := defaultrpc.NewRPCServer(app, module) //默认会创建一个本地的RPC
	if err != nil {
		logging.Warning("Dial: %s", err)
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
	var advt, host string
	var port int

	// check the advertise address first
	// if it exists then use it, otherwise
	// use the address
	if len(config.Advertise) > 0 {
		advt = config.Advertise
	} else {
		advt = config.Address
	}

	parts := strings.Split(advt, ":")
	if len(parts) > 1 {
		host = strings.Join(parts[:len(parts)-1], ":")
		port, _ = strconv.Atoi(parts[len(parts)-1])
	} else {
		host = parts[0]
	}

	addr, err := addr.Extract(host)
	if err != nil {
		return err
	}

	// register service
	node := &registry.Node{
		ID:       config.Name + "@" + config.ID,
		Address:  addr,
		Port:     port,
		Metadata: config.Metadata,
	}
	s.id = node.ID
	node.Metadata["server"] = s.String()
	node.Metadata["registry"] = config.Registry.String()

	s.RLock()
	// Maps are ordered randomly, sort the keys for consistency

	var endpoints []*registry.Endpoint

	s.RUnlock()

	service := &registry.Service{
		Name:      config.Name,
		Version:   config.Version,
		Nodes:     []*registry.Node{node},
		Endpoints: endpoints,
	}

	s.Lock()
	registered := s.registered
	s.Unlock()

	if !registered {
		logging.Info("Registering node: %s", node.ID)
	}

	// create registry options
	rOpts := []registry.RegisterOption{registry.RegisterTTL(config.RegisterTTL)}

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
	var advt, host string
	var port int

	// check the advertise address first
	// if it exists then use it, otherwise
	// use the address
	if len(config.Advertise) > 0 {
		advt = config.Advertise
	} else {
		advt = config.Address
	}

	parts := strings.Split(advt, ":")
	if len(parts) > 1 {
		host = strings.Join(parts[:len(parts)-1], ":")
		port, _ = strconv.Atoi(parts[len(parts)-1])
	} else {
		host = parts[0]
	}

	addr, err := addr.Extract(host)
	if err != nil {
		return err
	}

	node := &registry.Node{
		ID:      config.Name + "@" + config.ID,
		Address: addr,
		Port:    port,
	}

	service := &registry.Service{
		Name:    config.Name,
		Version: config.Version,
		Nodes:   []*registry.Node{node},
	}

	logging.Info("Deregistering node: %s", node.ID)
	if err := config.Registry.Deregister(service); err != nil {
		return err
	}

	s.Lock()

	if !s.registered {
		s.Unlock()
		return nil
	}

	s.registered = false

	s.Unlock()
	return nil
}

func (s *rpcServer) Start() error {
	//config := s.Options()

	//s.Lock()
	// swap address
	//addr := s.opts.Address
	//s.opts.Address = ts.Addr()
	//s.Unlock()
	return nil
}

func (s *rpcServer) Stop() error {
	if s.server != nil {
		logging.Info("RPCServer closeing id(%s)", s.id)
		err := s.server.Done()
		if err != nil {
			logging.Warning("RPCServer close fail id(%s) error(%s)", s.id, err)
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
