package defaultrpc

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	nats "github.com/nats-io/nats.go"
	logging "github.com/shihray/gserver/logging"
	module "github.com/shihray/gserver/module"
	mqrpc "github.com/shihray/gserver/rpc"
	rpcpb "github.com/shihray/gserver/rpc/pb"
	utils "github.com/shihray/gserver/utils"
)

type NatsClient struct {
	callinfos         *sync.Map
	cmutex            sync.Mutex //操作callinfos的鎖
	callbackQueueName string
	app               module.App
	done              chan error
	session           module.ServerSession
	isClose           bool
}

func NewNatsClient(app module.App, session module.ServerSession) (client *NatsClient, err error) {
	client = new(NatsClient)
	client.session = session
	client.app = app
	client.callinfos = new(sync.Map)
	client.callbackQueueName = nats.NewInbox()
	client.done = make(chan error)
	client.isClose = false
	go client.onRequestHandle()
	return client, nil
}

func (c *NatsClient) Delete(key string) (err error) {
	c.callinfos.Delete(key)
	return
}

func (c *NatsClient) CloseFch(fch chan rpcpb.ResultInfo) {
	defer utils.RecoverFunc()
	close(fch) // panic if ch is closed
}

func (c *NatsClient) Done() (err error) {
	//清理 callinfos 列表
	c.callinfos.Range(func(key, clinetCallInfo interface{}) bool {
		if clinetCallInfo != nil {
			//關閉管道
			c.CloseFch(clinetCallInfo.(ClinetCallInfo).call)
			//從Map中刪除
			c.callinfos.Delete(key)
		}
		return true
	})

	c.callinfos = nil
	c.done <- nil
	c.isClose = true
	return
}

// 消息請求
func (c *NatsClient) Call(callInfo mqrpc.CallInfo, callback chan rpcpb.ResultInfo) error {
	//var err error
	if c.callinfos == nil {
		return fmt.Errorf("AMQPClient is closed")
	}
	callInfo.RpcInfo.ReplyTo = c.callbackQueueName
	var correlationID = callInfo.RpcInfo.Cid

	clinetCallInfo := &ClinetCallInfo{
		correlationID: correlationID,
		call:          callback,
		timeout:       callInfo.RpcInfo.Expired,
	}
	c.callinfos.Store(correlationID, *clinetCallInfo)
	_, err := c.Marshal(&callInfo.RpcInfo)
	if err != nil {
		return err
	}
	return nil
}

// 消息請求 不需要回覆
func (c *NatsClient) CallNR(callInfo mqrpc.CallInfo) error {
	_, err := c.Marshal(&callInfo.RpcInfo)
	if err != nil {
		return err
	}
	return nil
}

// 接收應答信息
func (c *NatsClient) onRequestHandle() error {
	defer utils.RecoverFunc()
	subs, err := c.app.Transport().SubscribeSync(c.callbackQueueName)
	if err != nil {
		return err
	}

	go func() {
		<-c.done
		subs.Unsubscribe()
	}()

	for !c.isClose {
		m, err := subs.NextMsg(time.Minute)
		if err != nil && err == nats.ErrTimeout {
			continue
		} else if err != nil {
			logging.Error("NatsClient error with '%v'", err)
			continue
		}

		resultInfo, err := c.UnmarshalResult(m.Data)
		if err != nil {
			logging.Error("資料解析錯誤 %s", err)
		} else {
			correlationID := resultInfo.Cid
			clinetCallInfo, _ := c.callinfos.Load(correlationID)
			//刪除
			c.callinfos.Delete(correlationID)
			if clinetCallInfo != nil {
				clinetCallInfo.(ClinetCallInfo).call <- *resultInfo
				c.CloseFch(clinetCallInfo.(ClinetCallInfo).call)
			} else {
				logging.Warn("可能客戶端已超時了，但服務端處理完還給回調了 : [%s]", correlationID)
			}
		}
	}

	return nil
}

// 保存解碼後的數據，Value可以為任意數據類型
func (c *NatsClient) UnmarshalResult(data []byte) (*rpcpb.ResultInfo, error) {
	var resultInfo rpcpb.ResultInfo
	err := proto.Unmarshal(data, &resultInfo)
	if err != nil {
		return nil, err
	} else {
		return &resultInfo, err
	}
}

func (c *NatsClient) Unmarshal(data []byte) (*rpcpb.RPCInfo, error) {
	//保存解碼後的數據，Value可以為任意數據類型
	var rpcInfo rpcpb.RPCInfo
	err := proto.Unmarshal(data, &rpcInfo)
	if err != nil {
		return nil, err
	} else {
		return &rpcInfo, err
	}
}

// goroutine safe
func (c *NatsClient) Marshal(rpcInfo *rpcpb.RPCInfo) ([]byte, error) {
	b, err := proto.Marshal(rpcInfo)
	return b, err
}
