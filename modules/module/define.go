package basemodule

import (
	"context"

	"github.com/nats-io/nats.go"
	mqrpc "github.com/shihray/gserver/modules/rpc"
	"github.com/shihray/gserver/modules/utils/conf"
)

type ProtocolMarshal interface {
	GetData() []byte
}

type ServerSession interface {
	GetID() string
	GetName() string
	GetRpc() mqrpc.RPCClient
	GetApp() App
	Call(ctx context.Context, _func string, params ...interface{}) (interface{}, string)
	CallNR(_func string, params ...interface{}) (err error)
	CallArgs(ctx context.Context, _func string, ArgsType []string, args [][]byte) (interface{}, string)
	CallNRArgs(_func string, ArgsType []string, args [][]byte) (err error)
}

type Module interface {
	OnInit(app App, settings *conf.ModuleSettings)
	Run(closeSig chan bool)
	Version() string                             // 模塊版本
	GetType() string                             // 模塊類型
	OnAppConfigurationLoaded(app App)            // 當App初始化時調用，這個接口不管這個模塊是否在這個進程運行都會調用
	OnConfChanged(settings *conf.ModuleSettings) // 為以後動態服務發現做準備
	OnDestroy()
}

type RPCModule interface {
	context.Context
	Module
	GetServerID() string //模塊類型
	RpcInvoke(moduleType string, _func string, params ...interface{}) (interface{}, string)
	RpcInvokeNR(moduleType string, _func string, params ...interface{}) error
	RpcInvokeArgs(moduleType string, _func string, ArgsType []string, args [][]byte) (interface{}, string)
	RpcInvokeNRArgs(moduleType string, _func string, ArgsType []string, args [][]byte) error
	GetModuleSettings() (settings *conf.ModuleSettings)
	GetStatistical() (statistical string, err error)
	GetExecuting() int64
}

type App interface {
	Run(mods ...Module) error
	SetMapRoute(fn func(app App, route string) string) error
	Configure(settings conf.Config) error
	OnInit(settings conf.Config) error
	OnDestroy() error
	Options() Options
	Transport() *nats.Conn
	GetServerByID(id string) (ServerSession, error)
	GetServersByType(Type string) []ServerSession
	GetSettings() conf.Config //獲取配置信息
	RpcInvoke(module RPCModule, moduleType string, _func string, params ...interface{}) (interface{}, string)
	RpcInvokeNR(module RPCModule, moduleType string, _func string, params ...interface{}) error
	/**
	添加一個 自定義參數序列化接口
	gate,system 關鍵詞一被占用請使用其他名稱
	*/
	AddRPCSerialize(name string, Interface RPCSerialize) error

	GetRPCSerialize() map[string]RPCSerialize

	GetModuleInited() func(app App, module Module)

	OnConfigurationLoaded(func(app App)) error
	OnModuleInited(func(app App, module Module)) error
	OnStartup(func(app App)) error

	SetProtocolMarshal(protocolMarshal func(Trace string, Result interface{}, Error string) (ProtocolMarshal, string)) error
	/**
	與客戶端通信的協議包接口
	*/
	ProtocolMarshal(Trace string, Result interface{}, Error string) (ProtocolMarshal, string)
	NewProtocolMarshal(data []byte) ProtocolMarshal
	GetProcessID() string
	WorkDir() string
}

/**
rpc 自定義參數序列化接口
*/
type RPCSerialize interface {
	/**
	序列化 結構體-->[]byte
	param 需要序列化的參數值
	@return ptype 當能夠序列化這個值,並且正確解析為[]byte時 返回改值正確的類型,否則返回 ""即可
	@return p 解析成功得到的數據, 如果無法解析該類型,或者解析失敗 返回nil即可
	@return err 無法解析該類型,或者解析失敗 返回錯誤信息
	*/
	Serialize(param interface{}) (ptype string, p []byte, err error)

	/**
	反序列化 []byte-->結構體
	ptype 參數類型 與Serialize函數中ptype 對應
	b   參數的字節流
	@return param 解析成功得到的數據結構
	@return err 無法解析該類型,或者解析失敗 返回錯誤信息
	*/
	Deserialize(ptype string, b []byte) (param interface{}, err error)

	/**
	返回這個接口能夠處理的所有類型
	*/
	GetTypes() []string
}
