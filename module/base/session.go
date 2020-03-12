package basemodule

import (
	"context"

	module "github.com/shihray/gserver/module"
	"github.com/shihray/gserver/registry"
	mqrpc "github.com/shihray/gserver/rpc"
	defaultrpc "github.com/shihray/gserver/rpc/base"
)

func NewServerSession(app module.App, name string) (module.ServerSession, error) {
	session := &serverSession{
		name: name,
		app:  app,
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

func (c *serverSession) GetRpc() mqrpc.RPCClient {
	return c.Rpc
}

func (c *serverSession) GetApp() module.App {
	return c.app
}

/**
消息请求 需要回复
*/
func (c *serverSession) Call(ctx context.Context, _func string, params ...interface{}) (interface{}, string) {
	return c.Rpc.Call(ctx, _func, params...)
}

/**
消息请求 不需要回复
*/
func (c *serverSession) CallNR(_func string, params ...interface{}) (err error) {
	return c.Rpc.CallNR(_func, params...)
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
