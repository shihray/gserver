package basemodule

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	module "github.com/shihray/gserver/modules/module"
	"github.com/shihray/gserver/modules/server"
	"github.com/shihray/gserver/modules/service"
	"github.com/shihray/gserver/modules/utils"
	"github.com/shihray/gserver/modules/utils/conf"
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

func LoadStatisticalMethod(j string) map[string]*StatisticalMethod {
	sm := map[string]*StatisticalMethod{}
	err := json.Unmarshal([]byte(j), &sm)
	if err == nil {
		return sm
	} else {
		return nil
	}
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
	rwmutex     sync.RWMutex
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
		Name:     subclass.GetType(),
		ID:       utils.GenerateID().String(),
		Version:  subclass.Version(),
	}
	for _, o := range opt {
		o(&opts)
	}

	if len(opts.Name) == 0 {
		opt = append(opt, server.Name(subclass.GetType()))
	}

	if len(opts.ID) == 0 {
		opt = append(opt, server.ID(utils.GenerateID().String()))
	}

	if len(opts.Version) == 0 {
		opt = append(opt, server.Version(subclass.Version()))
	}

	server := server.NewServer(opt...)
	server.OnInit(subclass, app, settings)
	hostname, _ := os.Hostname()
	server.Options().Metadata["hostname"] = hostname
	server.Options().Metadata["pid"] = fmt.Sprintf("%v", os.Getpid())

	ctx, cancel := context.WithCancel(context.Background())
	m.exit = cancel
	m.service = service.NewService(
		service.Server(server),
		service.RegisterTTL(app.Options().RegisterTTL),
		service.RegisterInterval(app.Options().RegisterInterval),
		service.Context(ctx),
	)

	go m.service.Run()
	m.GetServer().SetListener(m)
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

func (m *BaseModule) RpcInvoke(moduleType string, _func string, params ...interface{}) (result interface{}, err string) {
	return m.App.RpcInvoke(m.GetSubclass(), moduleType, _func, params...)
}

func (m *BaseModule) RpcInvokeNR(moduleType string, _func string, params ...interface{}) (err error) {
	return m.App.RpcInvokeNR(m.GetSubclass(), moduleType, _func, params...)
}

func (m *BaseModule) RpcInvokeArgs(moduleType string, _func string, ArgsType []string, args [][]byte) (result interface{}, err string) {
	server, e := m.App.GetRouteServer(moduleType)
	if e != nil {
		err = e.Error()
		return
	}
	return server.CallArgs(nil, _func, ArgsType, args)
}

func (m *BaseModule) RpcInvokeNRArgs(moduleType string, _func string, ArgsType []string, args [][]byte) (err error) {
	server, err := m.App.GetRouteServer(moduleType)
	if err != nil {
		return
	}
	return server.CallNRArgs(_func, ArgsType, args)
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
	m.rwmutex.Lock()
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
	m.rwmutex.Unlock()
	if m.listener != nil {
		m.listener.OnTimeOut(fn, Expired)
	}
}

func (m *BaseModule) OnError(fn string, callInfo *mqrpc.CallInfo, err error) {
	m.rwmutex.Lock()
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
	m.rwmutex.Unlock()
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
func (m *BaseModule) OnComplete(fn string, callInfo *mqrpc.CallInfo, result *rpcpb.ResultInfo, exec_time int64) {
	m.rwmutex.Lock()
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
	m.rwmutex.Unlock()
	if m.listener != nil {
		m.listener.OnComplete(fn, callInfo, result, exec_time)
	}
}

func (m *BaseModule) GetExecuting() int64 {
	return 0
	//return m.GetServer().GetRPCServer().GetExecuting()
}

func (m *BaseModule) GetStatistical() (statistical string, err error) {
	m.rwmutex.Lock()
	//重置
	now := time.Now().UnixNano()
	for _, s := range m.statistical {
		s.EndTime = now
	}
	b, err := json.Marshal(m.statistical)
	if err == nil {
		statistical = string(b)
	}
	m.rwmutex.Unlock()
	return
}
