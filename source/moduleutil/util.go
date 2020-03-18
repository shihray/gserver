package moduleutil

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/shihray/gserver/logging"
	module "github.com/shihray/gserver/module"
	baseModule "github.com/shihray/gserver/module/base"
	registry "github.com/shihray/gserver/registry"
	mqrpc "github.com/shihray/gserver/rpc"
	conf "github.com/shihray/gserver/utils/conf"
)

type resultInfo struct {
	Error  interface{} // error data
	Result interface{} // result
}

func newOptions(opts ...module.Option) module.Options {
	// Parse input parameters
	confPath := flag.String("conf", "", "Server configuration file path")
	ProcessID := flag.String("pid", "develop", "Server ProcessID?")
	flag.Parse()

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

	opt := module.Options{
		WorkDir:          applicationDir,                                     // 工作路徑
		ProcessID:        *ProcessID,                                         // pid
		ConfPath:         fmt.Sprintf("%s/conf/config.json", applicationDir), // config file path
		Registry:         registry.DefaultRegistry,                           // 註冊器
		RegisterInterval: time.Millisecond * time.Duration(6000),             // 多久註冊一次
		RegisterTTL:      time.Millisecond * time.Duration(6500),             // 服務器存活時間
		Debug:            true,                                               // 初始化偵錯模式
	}
	for _, o := range opts {
		o(&opt)
	}

	//if opt.NatsPool == nil {
	//	size := 10
	//	pool, err := BaseRPC.New(nats.DefaultURL, size)
	//	if err != nil {
	//		logging.Error("Nats Pool connection Error ", err.Error())
	//	}
	//	opt.NatsPool = pool
	//}

	if opt.Nats == nil {
		nc, err := nats.Connect(nats.DefaultURL)
		if err != nil {
			logging.Error("Nats 無法取得連線: %s", err.Error())
		}
		opt.Nats = nc
	}

	if *confPath != "" {
		opt.ConfPath = *confPath
	}

	_, err = os.Open(opt.ConfPath)
	if err != nil {
		panic(fmt.Sprintf("config path error %v", err)) // 文件不存在
	}

	return opt
}

type ModuleUtil struct {
	version         string
	settings        conf.Config
	serverList      sync.Map
	opts            module.Options
	rpcSerializes   map[string]module.RPCSerialize
	mapRoute        func(app module.App, route string) string // 將RPC註冊到router上
	configLoaded    func(app module.App)
	startup         func(app module.App)
	moduleInited    func(app module.App, module module.Module)
	protocolMarshal func(Result interface{}, Error interface{}) (module.ProtocolMarshal, string)
	lock            sync.RWMutex
}

func NewApp(opts ...module.Option) module.App {
	options := newOptions(opts...)
	mu := new(ModuleUtil)
	mu.opts = options
	mu.rpcSerializes = map[string]module.RPCSerialize{}

	return mu
}

func (mu *ModuleUtil) OnInit(settings conf.Config) error {
	mu.lock.Lock()
	mu.settings = settings
	mu.lock.Unlock()
	return nil
}

func (mu *ModuleUtil) Configure(settings conf.Config) error {
	mu.lock.Lock()
	mu.settings = settings
	mu.lock.Unlock()
	return nil
}

func (mu *ModuleUtil) GetSettings() conf.Config {
	return mu.settings
}

func (mu *ModuleUtil) GetProcessID() string {
	return mu.opts.ProcessID
}

func (mu *ModuleUtil) Run(mods ...module.Module) error {
	f, err := os.Open(mu.opts.ConfPath)
	if err != nil {
		panic(fmt.Sprintf("config path error %v", err))
	}
	fmt.Printf("Server configuration path : %s\n", mu.opts.ConfPath)

	conf.LoadConfig(f.Name())
	mu.OnInit(conf.Conf)

	manager := baseModule.NewModuleManager()
	// register module to manager
	for i := 0; i < len(mods); i++ {
		mods[i].OnAppConfigurationLoaded(mu)
		manager.Register(mods[i])
	}
	mu.OnInit(mu.settings)
	manager.Init(mu, mu.opts.ProcessID)
	if mu.startup != nil {
		mu.startup(mu)
	}
	// close
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)
	sig := <-c

	wait := make(chan struct{})
	go func() {
		manager.Destroy()
		mu.OnDestroy()
		wait <- struct{}{}
	}()
	select {
	case <-wait:
		logging.Info("mqant closing down (signal: %v)", sig)
	}

	return nil
}

func (mu *ModuleUtil) SetMapRoute(fn func(app module.App, route string) string) error {
	mu.lock.Lock()
	mu.mapRoute = fn
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
func (mu *ModuleUtil) Watcher(s *registry.Service) {
	session, ok := mu.serverList.Load(s.ID)
	if ok && session != nil {
		session.(module.ServerSession).GetRpc().Done()
		mu.serverList.Delete(s.ID)
	}
}

func (mu *ModuleUtil) Registry() registry.Registry {
	return mu.opts.Registry
}

func (mu *ModuleUtil) GetRPCSerialize() map[string]module.RPCSerialize {
	return mu.rpcSerializes
}

// 移除已註銷的服務
func (mu *ModuleUtil) RemoveSutdownService(s *registry.Service) {
	session, ok := mu.serverList.Load(s.ID)
	if ok && session != nil {
		session.(module.ServerSession).GetRpc().Done()
		mu.serverList.Delete(s.ID)
	}
}

func (mu *ModuleUtil) OnDestroy() error {
	return nil
}

func (mu *ModuleUtil) GetServerByID(id string) (module.ServerSession, error) {
	services, err := mu.opts.Registry.GetService(id)
	if err != nil {
		logging.Warn("GetServersByType %v", err)
	}
	for _, service := range services {
		if _, ok := mu.serverList.Load(service.ID); !ok {
			s, err := baseModule.NewServerSession(mu, id, service)
			if err != nil {
				logging.Warn("NewServerSession %v", err)
			} else {
				s.SetService(service)
				mu.serverList.Store(service.ID, s)
			}
		}
	}
	if server, ok := mu.serverList.Load(id); !ok {
		return nil, errors.Errorf("%s Service Not Found", id)
	} else {
		return server.(module.ServerSession), nil
	}
}

func (mu *ModuleUtil) GetServersByType(id string) []module.ServerSession {
	sessions := make([]module.ServerSession, 0)
	services, err := mu.opts.Registry.GetService(id)
	if err != nil {
		logging.Warn("GetServersByType %v", err)
		return sessions
	}
	for _, service := range services {
		session, ok := mu.serverList.Load(service.ID)
		if !ok {
			s, err := baseModule.NewServerSession(mu, id, service)
			if err != nil {
				logging.Warn("NewServerSession %v", err)
			} else {
				mu.serverList.Store(service.ID, s)
				sessions = append(sessions, s)
			}
		} else {
			session.(module.ServerSession).SetService(service)
			sessions = append(sessions, session.(module.ServerSession))
		}
	}
	return sessions
}

func (mu *ModuleUtil) GetRouteServer(id string) (s module.ServerSession, err error) {
	if mu.mapRoute != nil {
		//进行一次路由转换
		id = mu.mapRoute(mu, id)
	}
	return mu.GetServerByID(id)
}

func (mu *ModuleUtil) RpcInvoke(module module.RPCModule, moduleID string, rpcInvokeResult *mqrpc.ResultInvokeST) (result interface{}, err string) {
	server, e := mu.GetServerByID(moduleID)
	if e != nil {
		err = e.Error()
		return
	}
	return server.Call(nil, rpcInvokeResult)
}

func (mu *ModuleUtil) RpcInvokeNR(module module.RPCModule, moduleID string, rpcInvokeResult *mqrpc.ResultInvokeST) (err error) {
	server, err := mu.GetServerByID(moduleID)
	if err != nil {
		return
	}
	return server.CallNR(rpcInvokeResult)
}

func (mu *ModuleUtil) GetModuleInit() func(app module.App, module module.Module) {
	return mu.moduleInited
}

func (mu *ModuleUtil) OnConfigLoaded(ifunc func(app module.App)) error {
	mu.configLoaded = ifunc
	return nil
}

func (mu *ModuleUtil) OnModuleInit(internalFunc func(app module.App, module module.Module)) error {
	mu.moduleInited = internalFunc
	return nil
}

func (mu *ModuleUtil) OnStartup(internalFunc func(app module.App)) error {
	mu.startup = internalFunc
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
	mu.protocolMarshal = protocolMarshal
	return nil
}

func (mu *ModuleUtil) ProtocolMarshal(Result interface{}, Error interface{}) (module.ProtocolMarshal, string) {
	if mu.protocolMarshal != nil {
		return mu.protocolMarshal(Result, Error)
	}
	r := &resultInfo{
		Error:  Error,
		Result: Result,
	}
	b, err := json.Marshal(r)
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
