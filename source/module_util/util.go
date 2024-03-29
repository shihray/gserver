package module_util

import (
	"context"
	"fmt"
	jsonIter "github.com/json-iterator/go"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	module "github.com/shihray/gserver/module"
	baseModule "github.com/shihray/gserver/module/base"
	ModuleRegistry "github.com/shihray/gserver/registry"
	mqRPC "github.com/shihray/gserver/rpc"
	CommonConf "github.com/shihray/gserver/utils/conf"
	log "github.com/z9905080/gloger"
	random "math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

type resultInfo struct {
	Error  interface{} // error data
	Result interface{} // result
}

func newOptions(opts ...module.Option) module.Options {
	var (
		applicationDir string = ""
		err            error  = nil
	)
	applicationDir, err = os.Getwd()
	if err != nil {
		file, _ := exec.LookPath(os.Args[0])
		ApplicationPath, _ := filepath.Abs(file)
		applicationDir, _ = filepath.Split(ApplicationPath)
	}
	confPath := applicationDir + "/conf/config.json"
	opt := module.Options{
		LogLevel:     int(log.DEBUG),                 // log 輸出等級
		LogMode:      int(log.Stdout),                // log 輸出模式
		ConfPath:     confPath,                       // config file path
		Registry:     ModuleRegistry.DefaultRegistry, // 註冊器
		RoutineCount: 1000,                           // Register Routine Channel length
	}
	for _, o := range opts {
		o(&opt)
	}

	if opt.Nats == nil {
		nc, err := nats.Connect(nats.DefaultURL)
		if err != nil {
			log.Error("Nats 無法取得連線:", err.Error())
		}
		opt.Nats = nc
	}

	_, err = os.Open(opt.ConfPath)
	if err != nil {
		panic(fmt.Sprintf("config path error %v", err)) // 文件不存在
	}

	return opt
}

type ModuleUtil struct {
	version                 string
	settings                CommonConf.Config
	serverList              sync.Map
	opts                    module.Options
	rpcSerializes           map[string]module.RPCSerialize
	mapRouteCallback        func(app module.App, route string) string // 將RPC註冊到router上
	configLoadCallback      func(app module.App)
	startupCallback         func(app module.App)
	moduleInitCallback      func(app module.App, module module.Module)
	protocolMarshalCallback func(Result interface{}, Error interface{}) (module.ProtocolMarshal, string)
	lock                    *sync.RWMutex
}

func NewApp(opts ...module.Option) module.App {
	options := newOptions(opts...)
	mu := new(ModuleUtil)
	mu.lock = new(sync.RWMutex)
	mu.opts = options
	mu.rpcSerializes = map[string]module.RPCSerialize{}

	return mu
}

func (mu *ModuleUtil) OnInit(settings CommonConf.Config) error {
	mu.lock.Lock()
	mu.settings = settings
	mu.lock.Unlock()
	return nil
}

func (mu *ModuleUtil) Configure(settings CommonConf.Config) error {
	mu.lock.Lock()
	mu.settings = settings
	mu.lock.Unlock()
	return nil
}

func (mu *ModuleUtil) GetSettings() CommonConf.Config {
	return mu.settings
}

func (mu *ModuleUtil) Run(mods ...module.Module) error {
	f, err := os.Open(mu.opts.ConfPath)
	if err != nil {
		panic(fmt.Sprintf("config path error %v", err))
	}
	log.Info("Server configuration path:", mu.opts.ConfPath)

	CommonConf.LoadConfig(f.Name())
	mu.OnInit(CommonConf.Conf)
	// 設定輸出Log等級
	log.SetCurrentLevel(log.Level(mu.opts.LogLevel))
	// 設定輸出模式 0:印出 1:寫入file
	log.SetLogMode(log.OutputMode(mu.opts.LogMode))

	manager := baseModule.NewModuleManager()
	// register module to manager
	for i := 0; i < len(mods); i++ {
		mods[i].OnAppConfigurationLoaded(mu)
		manager.Register(mods[i])
	}
	mu.OnInit(mu.settings)
	manager.Init(mu)
	if mu.startupCallback != nil {
		mu.startupCallback(mu)
	}

	wait := make(chan struct{})
	if mu.opts.ExitSignal == nil {
		// close
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)
		<-c

		go func() {
			manager.Destroy()
			mu.OnDestroy()
			wait <- struct{}{}
		}()
	} else {
		isStop := <-mu.opts.ExitSignal

		if isStop {
			go func() {
				manager.Destroy()
				mu.OnDestroy()
				wait <- struct{}{}
			}()
		}
	}
	select {
	case <-wait:
		log.Info("gserver closing down")
	}
	return nil
}

func (mu *ModuleUtil) SetMapRoute(fn func(app module.App, route string) string) error {
	mu.lock.Lock()
	mu.mapRouteCallback = fn
	mu.lock.Unlock()
	return nil
}

func (mu *ModuleUtil) AddRPCSerialize(name string, Interface module.RPCSerialize) error {
	if _, ok := mu.rpcSerializes[name]; ok {
		return fmt.Errorf("The name(%s) has been occupied", name)
	}
	mu.rpcSerializes[name] = Interface
	return nil
}

func (mu *ModuleUtil) Options() module.Options {
	return mu.opts
}

func (mu *ModuleUtil) Transport() *nats.Conn {
	return mu.opts.Nats
}

// 把註銷的服務serverSession刪除
func (mu *ModuleUtil) Watcher(s *ModuleRegistry.Service) {
	session, ok := mu.serverList.Load(s.ID)
	if ok && session != nil {
		session.(module.ServerSession).GetRpc().Done()
		mu.serverList.Delete(s.ID)
	}
}

func (mu *ModuleUtil) Registry() ModuleRegistry.Registry {
	return mu.opts.Registry
}

func (mu *ModuleUtil) GetRPCSerialize() map[string]module.RPCSerialize {
	return mu.rpcSerializes
}

// 移除已註銷的服務
func (mu *ModuleUtil) RemoveSutDownService(s *ModuleRegistry.Service) {
	session, ok := mu.serverList.Load(s.ID)
	if ok && session != nil {
		session.(module.ServerSession).GetRpc().Done()
		mu.Registry().Deregister(session.(module.ServerSession).GetService())
		mu.serverList.Delete(s.ID)
	}
}

func (mu *ModuleUtil) OnDestroy() error {
	return nil
}

func (mu *ModuleUtil) GetServiceList() ([]module.ServerSession, error) {
	sessions := make([]module.ServerSession, 0)
	serviceList, err := mu.opts.Registry.ListServices()
	if err != nil {
		return nil, err
	}
	for _, service := range serviceList {
		session, ok := mu.serverList.Load(service.ID)
		if !ok {
			s, err := baseModule.NewServerSession(mu, service.ID, service)
			if err != nil {
				log.Warn("NewServerSession", err)
			} else {
				mu.serverList.Store(service.ID, s)
				sessions = append(sessions, s)
			}
		} else {
			session.(module.ServerSession).SetService(service)
			sessions = append(sessions, session.(module.ServerSession))
		}
	}

	return sessions, nil
}

func (mu *ModuleUtil) GetServerByID(id string) (module.ServerSession, error) {
	session, isExist := mu.serverList.Load(id)
	if !isExist {
		serviceName := id
		s := strings.Split(id, "@")
		if len(s) == 2 {
			serviceName = s[0]
		} else {
			return nil, errors.Errorf("serverID is error %v", id)
		}

		sessionList, getListErr := mu.GetServersByType(serviceName)
		if getListErr != nil {
			return nil, getListErr
		}

		for _, serverSession := range sessionList {
			if serverSession.GetID() == id {
				return serverSession, nil
			}
		}
	} else {
		return session.(module.ServerSession), nil
	}
	return nil, errors.Errorf("%s Service Not Found", id)
}

func (mu *ModuleUtil) GetServersByType(typeName string) ([]module.ServerSession, error) {
	sessions := make([]module.ServerSession, 0)
	services, err := mu.opts.Registry.GetService(typeName)
	if err != nil {
		log.Warn("GetServersByType", err)
		return sessions, err
	}
	for _, service := range services {
		session, ok := mu.serverList.Load(service.ID)
		if !ok {
			s, newSessionErr := baseModule.NewServerSession(mu, service.ID, service)
			if newSessionErr != nil {
				log.Warn("NewServerSession", newSessionErr)
				return nil, newSessionErr
			} else {
				mu.serverList.Store(service.ID, s)
				sessions = append(sessions, s)
			}
		} else {
			session.(module.ServerSession).SetService(service)
			sessions = append(sessions, session.(module.ServerSession))
		}
	}
	return sessions, nil
}

func (mu *ModuleUtil) GetRandomServerByType(typeName string) (module.ServerSession, error) {
	var resp module.ServerSession

	servers, getErr := mu.GetServersByType(typeName)
	if getErr != nil {
		return resp, getErr
	}
	if len(servers) == 0 {
		return nil, errors.New("service not found")
	}
	seed := random.Intn(len(servers))
	resp = servers[seed]

	return resp, nil
}

func (mu *ModuleUtil) getRouteServer(filter string) (s module.ServerSession, err error) {
	sl := strings.Split(filter, "@")
	if len(sl) == 2 {
		moduleID := sl[1]
		if moduleID != "" {
			return mu.GetServerByID(filter)
		}
	}
	moduleType := sl[0]
	servers, e := mu.GetServersByType(moduleType)
	if e != nil {
		return nil, e
	}
	if len(servers) == 0 {
		return nil, errors.New("servers not found")
	}
	// 隨機選擇一組service
	seed := random.Intn(len(servers))
	server := servers[seed]
	return server, nil
}

func (mu *ModuleUtil) RpcInvoke(module module.RPCModule, moduleID string, rpcInvokeResult *mqRPC.ResultInvokeST, ctxList ...context.Context) (result interface{}, err string) {
	server, e := mu.getRouteServer(moduleID)
	if e != nil {
		err = e.Error()
		return
	}
	var ctx context.Context = nil
	if len(ctxList) > 0 {
		ctx = ctxList[0]
	}
	return server.Call(ctx, rpcInvokeResult)
}

func (mu *ModuleUtil) RpcInvokeNR(module module.RPCModule, moduleID string, rpcInvokeResult *mqRPC.ResultInvokeST) (err error) {
	server, e := mu.getRouteServer(moduleID)
	if e != nil {
		err = e
		return
	}
	return server.CallNR(rpcInvokeResult)
}

func (mu *ModuleUtil) GetModuleInit() func(app module.App, module module.Module) {
	return mu.moduleInitCallback
}

func (mu *ModuleUtil) OnConfigLoaded(i func(app module.App)) error {
	mu.configLoadCallback = i
	return nil
}

func (mu *ModuleUtil) OnModuleInit(internalFunc func(app module.App, module module.Module)) error {
	mu.moduleInitCallback = internalFunc
	return nil
}

func (mu *ModuleUtil) OnStartup(internalFunc func(app module.App)) error {
	mu.startupCallback = internalFunc
	return nil
}

// 回傳Client端格式設定
type protocolMarshalImp struct {
	data []byte
}

func (this *protocolMarshalImp) GetData() []byte {
	return this.data
}

func (mu *ModuleUtil) SetProtocolMarshal(protocolMarshal func(Result interface{}, Error interface{}) (module.ProtocolMarshal, string)) error {
	mu.protocolMarshalCallback = protocolMarshal
	return nil
}

func (mu *ModuleUtil) ProtocolMarshal(Result interface{}, Error interface{}) (module.ProtocolMarshal, string) {
	if mu.protocolMarshalCallback != nil {
		return mu.protocolMarshalCallback(Result, Error)
	}
	r := &resultInfo{
		Error:  Error,
		Result: Result,
	}
	b, err := jsonIter.Marshal(r)
	if err == nil {
		return mu.NewProtocolMarshal(b), ""
	} else {
		return nil, err.Error()
	}
}

func (mu *ModuleUtil) NewProtocolMarshal(data []byte) module.ProtocolMarshal {
	return &protocolMarshalImp{
		data: data,
	}
}
