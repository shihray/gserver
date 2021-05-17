package basemodule

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/shihray/gserver/registry"
	mqrpc "github.com/shihray/gserver/rpc"
	"github.com/shihray/gserver/utils/conf"
)

type ProtocolMarshal interface {
	GetData() []byte
}

type ServerSession interface {
	GetID() string
	GetName() string
	GetRpc() mqrpc.RPCClient
	GetApp() App
	GetService() *registry.Service
	SetService(s *registry.Service)
	Call(ctx context.Context, rpcInvokeResult *mqrpc.ResultInvokeST) (interface{}, string)
	CallNR(rpcInvokeResult *mqrpc.ResultInvokeST) (err error)
}

type Module interface {
	Version() string                             //模塊版本
	GetType() string                             //模塊類型
	OnAppConfigurationLoaded(app App)            //當App初始化時調用，這個接口不管這個模塊是否在這個進程運行都會調用
	OnConfChanged(settings *conf.ModuleSettings) //為以後動態服務發現做準備
	OnInit(app App, settings *conf.ModuleSettings)
	OnDestroy()
	GetApp() App
	Run(closeSig chan bool)
}

type RPCModule interface {
	Module
	GetServerID() string //模塊類型
	RpcInvoke(moduleType string, rpcInvokeResult *mqrpc.ResultInvokeST, ctxList ...context.Context) (interface{}, string)
	RpcInvokeNR(moduleType string, rpcInvokeResult *mqrpc.ResultInvokeST) error
	GetModuleSettings() (settings *conf.ModuleSettings)
	GetStatistical() (statistical string, err error)
	GetExecuting() int64
	CheckHeartbeat(typeName string)
}

type App interface {
	Run(mods ...Module) error
	SetMapRoute(fn func(app App, route string) string) error
	Configure(settings conf.Config) error
	OnInit(settings conf.Config) error
	OnDestroy() error
	Options() Options
	Transport() *nats.Conn
	Registry() registry.Registry
	GetServerByID(id string) (ServerSession, error)
	GetServersByType(name string) ([]ServerSession, error)
	GetRandomServerByType(name string) (ServerSession, error)
	GetServiceList() ([]ServerSession, error)

	GetSettings() conf.Config //獲取配置信息
	RpcInvoke(module RPCModule, moduleID string, rpcInvokeResult *mqrpc.ResultInvokeST, ctxList ...context.Context) (interface{}, string)
	RpcInvokeNR(module RPCModule, moduleID string, rpcInvokeResult *mqrpc.ResultInvokeST) error

	AddRPCSerialize(name string, Interface RPCSerialize) error

	GetRPCSerialize() map[string]RPCSerialize

	GetModuleInit() func(app App, module Module)

	OnConfigLoaded(func(app App)) error
	OnModuleInit(func(app App, module Module)) error
	OnStartup(func(app App)) error

	SetProtocolMarshal(protocolMarshal func(Result interface{}, Error interface{}) (ProtocolMarshal, string)) error
	ProtocolMarshal(Result interface{}, Error interface{}) (ProtocolMarshal, string) // 與客戶端通信的協議包接口
	NewProtocolMarshal(data []byte) ProtocolMarshal
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
