package mqrpc

import (
	"context"
	"reflect"

	rpcpb "github.com/shihray/gserver/rpc/pb"
)

type FunctionInfo struct {
	Function  reflect.Value
	Goroutine bool
}

type MQServer interface {
	Callback(callinfo CallInfo) error
}

type CallInfo struct {
	RpcInfo  rpcpb.RPCInfo
	Result   rpcpb.ResultInfo
	Props    map[string]interface{}
	ExecTime int64
	Agent    MQServer //代理者  AMQPServer / LocalServer 都继承 Callback(callinfo CallInfo)(error) 方法
}
type RPCListener interface {
	/**
	BeforeHandle会对请求做一些前置处理，如：检查当前玩家是否已登录，打印统计日志等。
	@session  可能为nil
	return error  当error不为nil时将直接返回改错误信息而不会再执行后续调用
	*/
	NoFoundFunction(fn string) (*FunctionInfo, error)
	BeforeHandle(fn string, callInfo *CallInfo) error
	OnTimeOut(fn string, Expired int64)
	OnError(fn string, callInfo *CallInfo, err error)
	/**
	fn 		方法名
	params		参数
	result		执行结果
	exec_time 	方法执行时间 单位为 Nano 纳秒  1000000纳秒等于1毫秒
	*/
	OnComplete(fn string, callInfo *CallInfo, result *rpcpb.ResultInfo, exec_time int64)
}

type GoroutineControl interface {
	Wait() error
	Finish()
}

type RPCServer interface {
	Addr() string
	SetListener(listener RPCListener)
	SetGoroutineControl(control GoroutineControl)
	GetExecuting() int64
	Register(id string, f interface{})
	RegisterGO(id string, f interface{})
	Done() (err error)
}

type RPCClient interface {
	Done() (err error)
	CallArgs(ctx context.Context, _func string, ArgsType []string, args [][]byte) (interface{}, string)
	CallNRArgs(_func string, ArgsType []string, args [][]byte) (err error)
	Call(ctx context.Context, rpcInvokeResult *ResultInvokeST) (interface{}, string)
	CallNR(rpcInvokeResult *ResultInvokeST) (err error)
}

type LocalClient interface {
	Done() error
	Call(callInfo CallInfo, callback chan rpcpb.ResultInfo) (err error)
	CallNR(callInfo CallInfo) (err error)
}
type LocalServer interface {
	IsClose() bool
	Write(callInfo CallInfo) error
	StopConsume() error
	Shutdown() (err error)
	Callback(callinfo CallInfo) error
}

// Marshaler is a simple encoding interface used for the broker/transport
// where headers are not supported by the underlying implementation.
type Marshaler interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
	String() string
}

// RPC Invoke Struct
type ResultInvokeST struct {
	funcName string        // function name
	params   []interface{} // params
}

// NewResultInvoke Create instance for RPC Invoke
func NewResultInvoke(funcName string, params ...interface{}) *ResultInvokeST {
	this := &ResultInvokeST{
		funcName: funcName,
		params:   params,
	}
	return this
}

// SetParams Set RPC Invoke Params
func (res *ResultInvokeST) SetParams(data ...interface{}) {
	res.params = data
}

// Get GetRPC Invoke Info
func (res *ResultInvokeST) Get() (string, []interface{}) {
	return res.funcName, res.params
}
