package defaultrpc

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	logging "github.com/shihray/gserver/logging"
	module "github.com/shihray/gserver/module"
	mqrpc "github.com/shihray/gserver/rpc"
	rpcpb "github.com/shihray/gserver/rpc/pb"
	argsutil "github.com/shihray/gserver/rpc/util"
	utils "github.com/shihray/gserver/utils"
	"github.com/shihray/gserver/utils/uuid"
)

const (
	ClientClose      string = "client close"
	DeadlineExceeded string = "deadline exceeded"
)

type RPCClient struct {
	app        module.App
	natsClient *NatsClient
}

func NewRPCClient(app module.App, session module.ServerSession) (mqrpc.RPCClient, error) {
	rpcClient := new(RPCClient)
	rpcClient.app = app
	natsClient, err := NewNatsClient(app, session)
	if err != nil {
		logging.Error("Nats RPC Client Create Error Dial: ", err)
		return nil, err
	}
	rpcClient.natsClient = natsClient
	return rpcClient, nil
}

func (c *RPCClient) Done() (err error) {
	if c.natsClient != nil {
		err = c.natsClient.Done()
	}
	return
}

func (c *RPCClient) CallArgs(ctx context.Context, internalFunc string, argsType []string, args [][]byte) (r interface{}, e string) {
	var correlationID = uuid.Rand().Hex()
	rpcInfo := &rpcpb.RPCInfo{
		Fn:       *proto.String(internalFunc),
		Reply:    *proto.Bool(true),
		Expired:  *proto.Int64((time.Now().UTC().Add(time.Second * time.Duration(c.app.GetSettings().Rpc.RpcExpired)).UnixNano()) / utils.Nano2Millisecond),
		Cid:      *proto.String(correlationID),
		Args:     args,
		ArgsType: argsType,
	}

	callInfo := &mqrpc.CallInfo{
		RpcInfo: *rpcInfo,
	}
	callback := make(chan rpcpb.ResultInfo, 1)

	err := c.natsClient.Call(*callInfo, callback)
	if err != nil {
		return nil, err.Error()
	}
	if ctx == nil {
		ctx, _ = context.WithTimeout(context.TODO(), time.Second*time.Duration(c.app.GetSettings().Rpc.RpcExpired))
	}
	select {
	case resultInfo, ok := <-callback:
		if !ok {
			return nil, ClientClose
		}
		result, err := argsutil.Bytes2Args(c.app, resultInfo.ResultType, resultInfo.Result)
		if err != nil {
			return nil, err.Error()
		}
		return result, resultInfo.Error
	case <-ctx.Done():
		c.closeCallbackChan(callback)
		c.natsClient.Delete(rpcInfo.Cid)
		return nil, DeadlineExceeded
	}
}

func (c *RPCClient) closeCallbackChan(ch chan rpcpb.ResultInfo) {
	defer utils.RecoverFunc()
	close(ch) // panic if ch is closed
}

func (c *RPCClient) CallNRArgs(ifunc string, argsType []string, args [][]byte) (err error) {
	var correlationID = uuid.Rand().Hex()
	rpcInfo := &rpcpb.RPCInfo{
		Fn:       *proto.String(ifunc),
		Reply:    *proto.Bool(false),
		Expired:  *proto.Int64((time.Now().UTC().Add(time.Second * time.Duration(c.app.GetSettings().Rpc.RpcExpired)).UnixNano()) / utils.Nano2Millisecond),
		Cid:      *proto.String(correlationID),
		Args:     args,
		ArgsType: argsType,
	}
	callInfo := &mqrpc.CallInfo{
		RpcInfo: *rpcInfo,
	}
	return c.natsClient.CallNR(*callInfo)
}

/**
消息请求 需要回复
*/
func (c *RPCClient) Call(ctx context.Context, rpcInvokeResult *mqrpc.ResultInvokeST) (interface{}, string) {
	funcName, params := rpcInvokeResult.Get()
	argsType := make([]string, len(params))
	args := make([][]byte, len(params))
	for k, param := range params {
		var err error = nil
		argsType[k], args[k], err = argsutil.ArgsTypeAnd2Bytes(c.app, param)
		if err != nil {
			return nil, fmt.Sprintf("args[%d] error %s", k, err.Error())
		}
	}
	start := time.Now()
	r, errstr := c.CallArgs(ctx, funcName, argsType, args)
	msg := fmt.Sprintf("RPC Call ServerID = %v Func = %v Elapsed = %v Result = %v ERROR = %v", c.natsClient.session.GetID(), funcName, time.Since(start), r, errstr)
	logging.Info(msg)

	return r, errstr
}

/**
消息请求 不需要回复
*/
func (c *RPCClient) CallNR(rpcInvokeResult *mqrpc.ResultInvokeST) (err error) {
	funcName, params := rpcInvokeResult.Get()
	argsType := make([]string, len(params))
	args := make([][]byte, len(params))
	for k, param := range params {
		argsType[k], args[k], err = argsutil.ArgsTypeAnd2Bytes(c.app, param)
		if err != nil {
			return fmt.Errorf("args[%d] error %s", k, err.Error())
		}
	}
	start := time.Now()
	err = c.CallNRArgs(funcName, argsType, args)
	msg := fmt.Sprintf("RPC Call ServerID = %v Func = %v Elapsed = %v Result = %v ERROR = %v", c.natsClient.session.GetID(), funcName, time.Since(start), err)
	logging.Info(msg)

	return err
}
