package defaultrpc

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	nats "github.com/nats-io/nats.go"
	logging "github.com/shihray/gserver/modules/logging"
	module "github.com/shihray/gserver/modules/module"
	mqrpc "github.com/shihray/gserver/modules/rpc"
	rpcpb "github.com/shihray/gserver/modules/rpc/pb"
	utils "github.com/shihray/gserver/modules/utils"
)

type NatsClient struct {
	//callinfos map[string]*ClinetCallInfo
	callinfos         *utils.BeeMap
	cmutex            sync.Mutex //操作callinfos的锁
	callbackqueueName string
	app               module.App
	done              chan error
	session           module.ServerSession
	isClose           bool
}

func NewNatsClient(app module.App, session module.ServerSession) (client *NatsClient, err error) {
	client = new(NatsClient)
	client.session = session
	client.app = app
	client.callinfos = utils.NewBeeMap()
	client.callbackqueueName = nats.NewInbox()
	client.done = make(chan error)
	client.isClose = false
	go client.on_request_handle()
	return client, nil
}

func (c *NatsClient) Delete(key string) (err error) {
	c.callinfos.Delete(key)
	return
}

func (c *NatsClient) CloseFch(fch chan rpcpb.ResultInfo) {
	defer func() {
		if recover() != nil {
			// close(ch) panic occur
		}
	}()
	close(fch) // panic if ch is closed
}

func (c *NatsClient) Done() (err error) {
	//关闭amqp链接通道
	//close(c.send_chan)
	//c.send_done<-nil

	//清理 callinfos 列表
	for key, clinetCallInfo := range c.callinfos.Items() {
		if clinetCallInfo != nil {
			//关闭管道
			c.CloseFch(clinetCallInfo.(ClinetCallInfo).call)
			//从Map中删除
			c.callinfos.Delete(key)
		}
	}
	c.callinfos = nil
	c.done <- nil
	c.isClose = true
	return
}

// 消息请求
func (c *NatsClient) Call(callInfo mqrpc.CallInfo, callback chan rpcpb.ResultInfo) error {
	//var err error
	if c.callinfos == nil {
		return fmt.Errorf("AMQPClient is closed")
	}
	callInfo.RpcInfo.ReplyTo = c.callbackqueueName
	var correlation_id = callInfo.RpcInfo.Cid

	clinetCallInfo := &ClinetCallInfo{
		correlation_id: correlation_id,
		call:           callback,
		timeout:        callInfo.RpcInfo.Expired,
	}
	c.callinfos.Set(correlation_id, *clinetCallInfo)
	body, err := c.Marshal(&callInfo.RpcInfo)
	if err != nil {
		return err
	}
	// return c.app.Transport().Publish(c.session.GetNode().Address, body)
	return nil
}

// 消息请求 不需要回复
func (c *NatsClient) CallNR(callInfo mqrpc.CallInfo) error {
	body, err := c.Marshal(&callInfo.RpcInfo)
	if err != nil {
		return err
	}
	// return c.app.Transport().Publish(c.session.GetNode().Address, body)
	return nil
}

/**
接收应答信息
*/
func (c *NatsClient) on_request_handle() error {
	defer func() {
		if r := recover(); r != nil {
			var rn = ""
			switch r.(type) {

			case string:
				rn = r.(string)
			case error:
				rn = r.(error).Error()
			}
			buf := make([]byte, 1024)
			l := runtime.Stack(buf, false)
			errstr := string(buf[:l])
			logging.Error(fmt.Sprintf("%s\n ----Stack----\n%s", rn, errstr))
			fmt.Println(errstr)
		}
	}()
	subs, err := c.app.Transport().SubscribeSync(c.callbackqueueName)
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
			fmt.Println(fmt.Sprintf("%v rpcclient error: %v", time.Now().String(), err.Error()))
			logging.Error(fmt.Sprintf("NatsClient error with '%v'", err))
			continue
		}

		resultInfo, err := c.UnmarshalResult(m.Data)
		if err != nil {
			logging.Error(fmt.Sprintf("Unmarshal faild", err))
		} else {
			correlation_id := resultInfo.Cid
			clinetCallInfo := c.callinfos.Get(correlation_id)
			//删除
			c.callinfos.Delete(correlation_id)
			if clinetCallInfo != nil {
				clinetCallInfo.(ClinetCallInfo).call <- *resultInfo
				c.CloseFch(clinetCallInfo.(ClinetCallInfo).call)
			} else {
				//可能客户端已超时了，但服务端处理完还给回调了
				logging.Warning(fmt.Sprintf("rpc callback no found : [%s]", correlation_id))
			}
		}
	}

	return nil
}

// 保存解码后的数据，Value可以为任意数据类型
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
	//fmt.Println(msg)
	//保存解码后的数据，Value可以为任意数据类型
	var rpcInfo rpcpb.RPCInfo
	err := proto.Unmarshal(data, &rpcInfo)
	if err != nil {
		return nil, err
	} else {
		return &rpcInfo, err
	}

	panic("bug")
}

// goroutine safe
func (c *NatsClient) Marshal(rpcInfo *rpcpb.RPCInfo) ([]byte, error) {
	//map2:= structs.Map(callInfo)
	b, err := proto.Marshal(rpcInfo)
	return b, err
}
