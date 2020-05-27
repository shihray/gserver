package defaultrpc

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	nats "github.com/nats-io/nats.go"
	module "github.com/shihray/gserver/module"
	mqrpc "github.com/shihray/gserver/rpc"
	rpcPB "github.com/shihray/gserver/rpc/pb"
	utils "github.com/shihray/gserver/utils"
	log "github.com/z9905080/gloger"
)

type NatsClient struct {
	callinfos         *sync.Map
	cmutex            *sync.RWMutex
	callbackQueueName string
	app               module.App
	done              chan error
	session           module.ServerSession
	isClose           bool
}

func NewNatsClient(app module.App, session module.ServerSession) (client *NatsClient, err error) {
	client = new(NatsClient)
	client.cmutex = new(sync.RWMutex)
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

func (c *NatsClient) CloseFch(fch chan rpcPB.ResultInfo) {
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
func (c *NatsClient) Call(callInfo mqrpc.CallInfo, callback chan rpcPB.ResultInfo) error {
	if c.callinfos == nil {
		return fmt.Errorf("AMQPClient is closed")
	}
	callInfo.RpcInfo.ReplyTo = c.callbackQueueName
	var correlationID = callInfo.RpcInfo.Cid

	clientCallInfo := &ClinetCallInfo{
		correlationID: correlationID,
		call:          callback,
		timeout:       callInfo.RpcInfo.Expired,
	}
	c.callinfos.Store(correlationID, *clientCallInfo)
	body, err := c.Marshal(&callInfo.RpcInfo)
	if err != nil {
		return err
	}
	return c.app.Transport().Publish(c.session.GetService().Address, body)
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
			log.Error("NatsClient error with ", err.Error())
			continue
		}

		resultInfo, err := c.UnmarshalResult(m.Data)
		if err != nil {
			log.Error("資料解析錯誤", err.Error())
		} else {
			correlationID := resultInfo.Cid
			clientCallInfo, _ := c.callinfos.Load(correlationID)
			//刪除
			c.callinfos.Delete(correlationID)
			if clientCallInfo != nil {
				clientCallInfo.(ClinetCallInfo).call <- *resultInfo
				c.CloseFch(clientCallInfo.(ClinetCallInfo).call)
			} else {
				log.Warn("可能客戶端已超時了，但服務端處理完還給回調了:", correlationID)
			}
		}
	}

	return nil
}

// 保存解碼後的數據，Value可以為任意數據類型
func (c *NatsClient) UnmarshalResult(data []byte) (*rpcPB.ResultInfo, error) {
	var resultInfo rpcPB.ResultInfo
	err := proto.Unmarshal(data, &resultInfo)
	if err != nil {
		return nil, err
	} else {
		return &resultInfo, err
	}
}

func (c *NatsClient) Unmarshal(data []byte) (*rpcPB.RPCInfo, error) {
	//保存解碼後的數據，Value可以為任意數據類型
	var rpcInfo rpcPB.RPCInfo
	err := proto.Unmarshal(data, &rpcInfo)
	if err != nil {
		return nil, err
	} else {
		return &rpcInfo, err
	}
}

// goroutine safe
func (c *NatsClient) Marshal(rpcInfo *rpcPB.RPCInfo) ([]byte, error) {
	b, err := proto.Marshal(rpcInfo)
	return b, err
}
