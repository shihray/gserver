package defaultrpc

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	logging "github.com/shihray/gserver/modules/logging"
	module "github.com/shihray/gserver/modules/module"
	mqrpc "github.com/shihray/gserver/modules/rpc"
	rpcpb "github.com/shihray/gserver/modules/rpc/pb"
	argsutil "github.com/shihray/gserver/modules/rpc/util"
	"github.com/shihray/gserver/modules/utils/uuid"
)

type RPCClient struct {
	app         module.App
	nats_client *NatsClient
}

func NewRPCClient(app module.App, session module.ServerSession) (mqrpc.RPCClient, error) {
	rpc_client := new(RPCClient)
	rpc_client.app = app
	nats_client, err := NewNatsClient(app, session)
	if err != nil {
		logging.Error("Dial: %s", err)
		return nil, err
	}
	rpc_client.nats_client = nats_client
	return rpc_client, nil
}

func (c *RPCClient) Done() (err error) {
	if c.nats_client != nil {
		err = c.nats_client.Done()
	}
	return
}

func (c *RPCClient) CallArgs(ctx context.Context, _func string, ArgsType []string, args [][]byte) (r interface{}, e string) {
	start := time.Now()
	var correlation_id = uuid.Rand().Hex()
	rpcInfo := &rpcpb.RPCInfo{
		Fn:       *proto.String(_func),
		Reply:    *proto.Bool(true),
		Expired:  *proto.Int64((time.Now().UTC().Add(time.Second * time.Duration(c.app.GetSettings().Rpc.RpcExpired)).UnixNano()) / 1000000),
		Cid:      *proto.String(correlation_id),
		Args:     args,
		ArgsType: ArgsType,
	}

	callInfo := &mqrpc.CallInfo{
		RpcInfo: *rpcInfo,
	}
	callback := make(chan rpcpb.ResultInfo, 1)

	var err error

	err = c.nats_client.Call(*callInfo, callback)
	if err != nil {
		return nil, err.Error()
	}
	if ctx == nil {
		ctx, _ = context.WithTimeout(context.TODO(), time.Second*time.Duration(c.app.GetSettings().Rpc.RpcExpired))
	}
	select {
	case resultInfo, ok := <-callback:
		if !ok {
			return nil, "client closed"
		}
		result, err := argsutil.Bytes2Args(c.app, resultInfo.ResultType, resultInfo.Result)
		if err != nil {
			return nil, err.Error()
		}
		return result, resultInfo.Error
	case <-ctx.Done():
		c.close_callback_chan(callback)
		c.nats_client.Delete(rpcInfo.Cid)
		return nil, "deadline exceeded"
	}
}

func (c *RPCClient) close_callback_chan(ch chan rpcpb.ResultInfo) {
	defer RecoverFunc()
	close(ch) // panic if ch is closed
}

func (c *RPCClient) CallNRArgs(_func string, ArgsType []string, args [][]byte) (err error) {
	var correlation_id = uuid.Rand().Hex()
	rpcInfo := &rpcpb.RPCInfo{
		Fn:       *proto.String(_func),
		Reply:    *proto.Bool(false),
		Expired:  *proto.Int64((time.Now().UTC().Add(time.Second * time.Duration(c.app.GetSettings().Rpc.RpcExpired)).UnixNano()) / 1000000),
		Cid:      *proto.String(correlation_id),
		Args:     args,
		ArgsType: ArgsType,
	}
	callInfo := &mqrpc.CallInfo{
		RpcInfo: *rpcInfo,
	}
	return c.nats_client.CallNR(*callInfo)
}

/**
消息请求 需要回复
*/
func (c *RPCClient) Call(ctx context.Context, _func string, params ...interface{}) (interface{}, string) {
	var ArgsType []string = make([]string, len(params))
	var args [][]byte = make([][]byte, len(params))
	for k, param := range params {
		var err error = nil
		ArgsType[k], args[k], err = argsutil.ArgsTypeAnd2Bytes(c.app, param)
		if err != nil {
			return nil, fmt.Sprintf("args[%d] error %s", k, err.Error())
		}
	}
	start := time.Now()
	r, errstr := c.CallArgs(ctx, _func, ArgsType, args)
	logging.Error(span, "RPC Call ServerID = %v Func = %v Elapsed = %v Result = %v ERROR = %v", c.nats_client.session.GetID(), _func, time.Since(start), r, errstr)

	return r, errstr
}

/**
消息请求 不需要回复
*/
func (c *RPCClient) CallNR(_func string, params ...interface{}) (err error) {
	var ArgsType []string = make([]string, len(params))
	var args [][]byte = make([][]byte, len(params))
	for k, param := range params {
		ArgsType[k], args[k], err = argsutil.ArgsTypeAnd2Bytes(c.app, param)
		if err != nil {
			return fmt.Errorf("args[%d] error %s", k, err.Error())
		}
	}
	start := time.Now()
	err = c.CallNRArgs(_func, ArgsType, args)
	logging.Error(span, "RPC CallNR ServerID = %v Func = %v Elapsed = %v ERROR = %v", c.nats_client.session.GetID(), _func, time.Since(start), err)

	return err
}
