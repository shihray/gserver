package basemodule

import (
	"context"

	module "github.com/shihray/gserver/module"
	"github.com/shihray/gserver/registry"
	mqrpc "github.com/shihray/gserver/rpc"
	defaultrpc "github.com/shihray/gserver/rpc/base"
)

func NewServerSession(app module.App, name string, s *registry.Service) (module.ServerSession, error) {
	session := &serverSession{
		name:    name,
		app:     app,
		service: s,
	}
	rpc, err := defaultrpc.NewRPCClient(app, session)
	if err != nil {
		return nil, err
	}
	session.Rpc = rpc
	return session, err
}

type serverSession struct {
	service *registry.Service // 註冊服務
	name    string            // 名稱
	Rpc     mqrpc.RPCClient
	app     module.App
}

func (c *serverSession) GetID() string {
	return c.service.ID
}

func (c *serverSession) GetName() string {
	return c.name
}

func (c *serverSession) GetService() *registry.Service {
	return c.service
}

func (c *serverSession) SetService(s *registry.Service) {
	c.service = s
}

func (c *serverSession) GetRpc() mqrpc.RPCClient {
	return c.Rpc
}

func (c *serverSession) GetApp() module.App {
	return c.app
}

/**
消息请求 需要回复
*/
func (c *serverSession) Call(ctx context.Context, rpcInvokeResult *mqrpc.ResultInvokeST) (interface{}, string) {
	return c.Rpc.Call(ctx, rpcInvokeResult)
}

/**
消息请求 不需要回复
*/
func (c *serverSession) CallNR(rpcInvokeResult *mqrpc.ResultInvokeST) (err error) {
	return c.Rpc.CallNR(rpcInvokeResult)
}

/**
消息请求 需要回复
*/
func (c *serverSession) CallArgs(ctx context.Context, _func string, ArgsType []string, args [][]byte) (interface{}, string) {
	return c.Rpc.CallArgs(ctx, _func, ArgsType, args)
}

/**
消息请求 不需要回复
*/
func (c *serverSession) CallNRArgs(_func string, ArgsType []string, args [][]byte) (err error) {
	return c.Rpc.CallNRArgs(_func, ArgsType, args)
}
