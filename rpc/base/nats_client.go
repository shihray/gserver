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
	subs              *nats.Subscription
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
	correlationID := callInfo.RpcInfo.Cid

	clientCallInfo := &ClinetCallInfo{
		correlationID: correlationID,
		call:          callback,
		timeout:       callInfo.RpcInfo.Expired,
	}
	c.callinfos.Store(correlationID, *clientCallInfo)

	resp := ""
	for _, arg := range callInfo.RpcInfo.Args {
		resp += " " + string(arg)
	}
	log.DebugF("儲存發送消息 cid:%v, Args:%v", correlationID, resp)

	body, err := c.Marshal(&callInfo.RpcInfo)
	if err != nil {
		return err
	}
	return c.app.Transport().Publish(c.session.GetService().Address, body)
}

// 消息請求 不需要回覆
func (c *NatsClient) CallNR(callInfo mqrpc.CallInfo) error {
	body, err := c.Marshal(&callInfo.RpcInfo)
	if err != nil {
		return err
	}
	return c.app.Transport().Publish(c.session.GetService().Address, body)
}

// 接收應答信息
func (c *NatsClient) onRequestHandle() (err error) {
	defer utils.RecoverFunc()

	log.DebugF("callbackQueueName : %v", c.callbackQueueName)

	c.subs, err = c.app.Transport().SubscribeSync(c.callbackQueueName)
	if err != nil {
		return err
	}

	go func() {
		<-c.done
		c.subs.Unsubscribe()
	}()

	for !c.isClose {
		m, msgErr := c.subs.NextMsg(10 * time.Second)
		if msgErr != nil && msgErr == nats.ErrTimeout {

			if !c.subs.IsValid() {
				// 訂閱已關閉，需要重新訂閱
				c.subs, msgErr = c.app.Transport().SubscribeSync(c.callbackQueueName)
				if msgErr != nil {
					log.ErrorF("NatsClient SubscribeSync[1] error with '%v'", msgErr)
					continue
				}
				log.WarnF("NatsClient SubscribeSync[1] %v", c.callbackQueueName)
			}

			continue
		} else if msgErr != nil {
			log.Error("NatsClient error with ", msgErr.Error())
			if !c.subs.IsValid() {
				// 訂閱已關閉，需要重新訂閱
				c.subs, err = c.app.Transport().SubscribeSync(c.callbackQueueName)
				if err != nil {
					log.ErrorF("NatsClient SubscribeSync[2] error with '%v'", err)
					continue
				}
				log.WarnF("NatsClient SubscribeSync[2] %v", c.callbackQueueName)
			}
			continue
		}

		resultInfo, msgErr := c.UnmarshalResult(m.Data)
		if msgErr != nil {
			log.Error("Unmarshal failed", msgErr.Error())
		} else {
			correlationID := resultInfo.Cid
			clientCallInfo, ok := c.callinfos.Load(correlationID)
			if !ok {
				log.Warn("NatsClient can't found map key:", correlationID)
			}

			log.Debug("onRequestHandle接收：", correlationID, "result: ", string(resultInfo.Result), " Error: ", resultInfo.Error)

			//刪除
			c.callinfos.Delete(correlationID)
			if clientCallInfo != nil {
				if clientCallInfo.(ClinetCallInfo).call != nil {
					clientCallInfo.(ClinetCallInfo).call <- *resultInfo
					c.CloseFch(clientCallInfo.(ClinetCallInfo).call)
				}
			} else {
				log.Warn("rpc callback no found:", correlationID)
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
