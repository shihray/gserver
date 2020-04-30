package basemodule

import (
	"context"
	"encoding/json"
	"fmt"
	ModuleRegistry "github.com/shihray/gserver/registry"
	defaultRPC "github.com/shihray/gserver/rpc/base"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/shihray/gserver/logging"
	module "github.com/shihray/gserver/module"
	mqrpc "github.com/shihray/gserver/rpc"
	rpcPB "github.com/shihray/gserver/rpc/pb"
	"github.com/shihray/gserver/server"
	"github.com/shihray/gserver/service"
	"github.com/shihray/gserver/utils"
	"github.com/shihray/gserver/utils/conf"
)

type StatisticalMethod struct {
	Name        string // 方法名
	StartTime   int64  // 開始時間
	EndTime     int64  // 結束時間
	MinExecTime int64  // 最短執行時間
	MaxExecTime int64  // 最長執行時間
	ExecTotal   int    // 執行總次數
	ExecTimeout int    // 執行超時次數
	ExecSuccess int    // 執行成功次數
	ExecFailure int    // 執行錯誤次數
}

type BaseModule struct {
	context.Context
	exit        context.CancelFunc
	App         module.App
	subclass    module.RPCModule
	settings    *conf.ModuleSettings
	service     service.Service
	listener    mqrpc.RPCListener
	statistical map[string]*StatisticalMethod //統計
	rwMutex     sync.RWMutex
	routineLock chan bool
}

func (m *BaseModule) GetServerID() string {
	//很關鍵,需要與配置文件中的Module配置對應
	return m.service.Server().ID()
}

func (m *BaseModule) GetApp() module.App {
	return m.App
}

func (m *BaseModule) GetSubclass() module.RPCModule {
	return m.subclass
}

func (m *BaseModule) GetServer() server.Server {
	return m.service.Server()
}

func (m *BaseModule) OnConfChanged(settings *conf.ModuleSettings) {

}

//當App初始化時調用，這個接口不管這個模塊是否在這個進程運行都會調用
func (m *BaseModule) OnAppConfigurationLoaded(app module.App) {
	m.App = app
}

func (m *BaseModule) OnInit(subclass module.RPCModule, app module.App, settings *conf.ModuleSettings, opt ...server.Option) {
	//初始化模塊
	m.App = app
	m.subclass = subclass
	m.settings = settings
	m.statistical = map[string]*StatisticalMethod{}
	//創建一個遠程調用的RPC
	opts := server.Options{
		Metadata: map[string]string{},
	}
	for _, o := range opt {
		o(&opts)
	}

	if opts.Registry == nil {
		opt = append(opt, server.Registry(app.Registry()))
	}

	if len(opts.Name) == 0 {
		opt = append(opt, server.Name(subclass.GetType()))
	}

	if len(opts.ID) == 0 {
		opt = append(opt, server.ID(utils.GenerateID().String()))
	}

	if opts.RoutineCount == 0 {
		opt = append(opt, server.RoutineCount(app.Options().RoutineCount))
	}
	//
	m.CheckHeartbeat(subclass.GetType())

	rpcServer := server.NewServer(opt...)
	_ = rpcServer.OnInit(subclass, app, settings)
	hostname, _ := os.Hostname()
	rpcServer.Options().Metadata["hostname"] = hostname
	rpcServer.Options().Metadata["pid"] = fmt.Sprintf("%v", os.Getpid())
	// heartbeat function
	rpcServer.Register("HB", func(m map[string]interface{}) (string, string) {
		return "Done", ""
	})

	ctx, cancel := context.WithCancel(context.Background())
	m.exit = cancel
	m.service = service.NewService(
		service.Server(rpcServer),
		service.Context(ctx),
	)

	go m.service.Run()
	m.GetServer().SetListener(m)

	// implement go routine control channel
	m.routineLock = make(chan bool, app.Options().RoutineCount)
	m.GetServer().SetGoroutineControl(m)
}

// Start when create new routine add lock
func (m *BaseModule) Start(num int) {
	for i := 0; i < num; i++ {
		m.routineLock <- true
	}

}

// Finish pop chan when routine finish
func (m *BaseModule) Finish(num int) {
	for i := 0; i < num; i++ {
		<-m.routineLock
	}
}

func (m *BaseModule) OnDestroy() {
	m.exit()
	m.GetServer().OnDestroy()
}

func (m *BaseModule) SetListener(listener mqrpc.RPCListener) {
	m.listener = listener
}

func (m *BaseModule) GetModuleSettings() *conf.ModuleSettings {
	return m.settings
}

// GetRandomServiceID 取得隨機 module ID
func (m *BaseModule) GetRandomServiceID(typeName string) (result string, err error) {
	services, getServerErr := m.GetServersByType(typeName)
	if getServerErr != nil {
		return "", getServerErr
	}
	if len(services) == 0 {
		return "", ServiceNotFound.Error()
	}
	index := rand.Intn(len(services))
	return services[index].GetName(), nil
}

func (m *BaseModule) GetServersByType(typeName string) (s []module.ServerSession, err error) {
	return m.App.GetServersByType(typeName)
}

func (m *BaseModule) RpcInvoke(moduleType string, rpcInvokeResult *mqrpc.ResultInvokeST) (result interface{}, err string) {
	return m.App.RpcInvoke(m.GetSubclass(), moduleType, rpcInvokeResult)
}

func (m *BaseModule) RpcInvokeNR(moduleType string, rpcInvokeResult *mqrpc.ResultInvokeST) (err error) {
	return m.App.RpcInvokeNR(m.GetSubclass(), moduleType, rpcInvokeResult)
}

func (m *BaseModule) NoFoundFunction(fn string) (*mqrpc.FunctionInfo, error) {
	if m.listener != nil {
		return m.listener.NoFoundFunction(fn)
	}
	return nil, errors.Errorf("Remote function(%s) not found", fn)
}

func (m *BaseModule) BeforeHandle(fn string, callInfo *mqrpc.CallInfo) error {
	if m.listener != nil {
		return m.listener.BeforeHandle(fn, callInfo)
	}
	return nil
}

func (m *BaseModule) OnTimeOut(fn string, Expired int64) {
	m.rwMutex.Lock()
	if statisticalMethod, ok := m.statistical[fn]; ok {
		statisticalMethod.ExecTimeout++
		statisticalMethod.ExecTotal++
	} else {
		statisticalMethod := &StatisticalMethod{
			Name:        fn,
			StartTime:   time.Now().UnixNano(),
			ExecTimeout: 1,
			ExecTotal:   1,
		}
		m.statistical[fn] = statisticalMethod
	}
	m.rwMutex.Unlock()
	if m.listener != nil {
		m.listener.OnTimeOut(fn, Expired)
	}
}

func (m *BaseModule) OnError(fn string, callInfo *mqrpc.CallInfo, err error) {
	m.rwMutex.Lock()
	if statisticalMethod, ok := m.statistical[fn]; ok {
		statisticalMethod.ExecFailure++
		statisticalMethod.ExecTotal++
	} else {
		statisticalMethod := &StatisticalMethod{
			Name:        fn,
			StartTime:   time.Now().UnixNano(),
			ExecFailure: 1,
			ExecTotal:   1,
		}
		m.statistical[fn] = statisticalMethod
	}
	m.rwMutex.Unlock()
	if m.listener != nil {
		m.listener.OnError(fn, callInfo, err)
	}
}

/**
fn 		    方法名
params		參數
result		執行結果
exec_time 	方法執行時間 單位為 Nano 納秒  1000000納秒等於1毫秒
*/
func (m *BaseModule) OnComplete(fn string, callInfo *mqrpc.CallInfo, result *rpcPB.ResultInfo, exec_time int64) {
	m.rwMutex.Lock()
	if statisticalMethod, ok := m.statistical[fn]; ok {
		statisticalMethod.ExecSuccess++
		statisticalMethod.ExecTotal++
		if statisticalMethod.MinExecTime > exec_time {
			statisticalMethod.MinExecTime = exec_time
		}
		if statisticalMethod.MaxExecTime < exec_time {
			statisticalMethod.MaxExecTime = exec_time
		}
	} else {
		statisticalMethod := &StatisticalMethod{
			Name:        fn,
			StartTime:   time.Now().UnixNano(),
			ExecSuccess: 1,
			ExecTotal:   1,
			MaxExecTime: exec_time,
			MinExecTime: exec_time,
		}
		m.statistical[fn] = statisticalMethod
	}
	m.rwMutex.Unlock()
	if m.listener != nil {
		m.listener.OnComplete(fn, callInfo, result, exec_time)
	}
}

func (m *BaseModule) GetExecuting() int64 {
	return 0
}

func (m *BaseModule) GetStatistical() (statistical string, err error) {
	m.rwMutex.Lock()
	//重置
	now := time.Now().UnixNano()
	for _, s := range m.statistical {
		s.EndTime = now
	}
	b, err := json.Marshal(m.statistical)
	if err == nil {
		statistical = string(b)
	}
	m.rwMutex.Unlock()
	return
}

// 檢查在線列表心跳
func (m *BaseModule) CheckHeartbeat(typeName string) {
	services, err := m.GetServersByType(typeName)
	if err != nil {
		log.Debug("GetServersByType Error", err.Error())
	}
	for _, session := range services {
		st := mqrpc.NewResultInvoke("HB", nil)
		go func(s module.ServerSession) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			if _, callServerErr := s.Call(ctx, st); callServerErr == defaultRPC.DeadlineExceeded || callServerErr == defaultRPC.ClientClose {
				if errOfDeregister := ModuleRegistry.Deregister(s.GetService()); errOfDeregister != nil {
					log.Debug("Heartbeat Error ", errOfDeregister)
				}
			}
		}(session)
	}
}
