package defaultrpc

import (
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats.go"
	module "github.com/shihray/gserver/module"
	mqrpc "github.com/shihray/gserver/rpc"
	rpcPB "github.com/shihray/gserver/rpc/pb"
	"github.com/shihray/gserver/utils"
	log "github.com/z9905080/gloger"
)

type NatsServer struct {
	callChan chan mqrpc.CallInfo
	addr     string
	app      module.App
	server   *RPCServer
	done     chan error
	isClose  bool
}

func NewNatsServer(app module.App, s *RPCServer) (*NatsServer, error) {
	server := new(NatsServer)
	server.server = s
	server.done = make(chan error)
	server.isClose = false
	server.app = app
	server.addr = nats.NewInbox()
	go server.onRequestHandle()
	return server, nil
}

// 取得Nats節點地址
func (s *NatsServer) Addr() string {
	return s.addr
}

// 註銷消息隊列
func (s *NatsServer) Shutdown() (err error) {
	s.done <- nil
	s.isClose = true
	return
}

// 回傳函數
func (s *NatsServer) Callback(callinfo mqrpc.CallInfo) error {
	body, _ := s.MarshalResult(callinfo.Result)
	replyTo := callinfo.Props["ReplyTo"].(string)
	return s.app.Transport().Publish(replyTo, body)
}

// 接收請求信息
func (s *NatsServer) onRequestHandle() error {
	defer utils.RecoverFunc()
	subs, err := s.app.Transport().SubscribeSync(s.addr)
	if err != nil {
		return err
	}

	go func() {
		<-s.done
		subs.Unsubscribe()
	}()

	for !s.isClose {
		m, err := subs.NextMsg(time.Minute)
		if err != nil && err == nats.ErrTimeout {
			continue
		} else if err != nil {
			log.Error("NatsServer error:", err)
			continue
		}

		rpcInfo, err := s.Unmarshal(m.Data)
		if err == nil {
			callInfo := &mqrpc.CallInfo{
				RpcInfo: *rpcInfo,
			}
			callInfo.Props = map[string]interface{}{
				"ReplyTo": rpcInfo.ReplyTo,
			}
			callInfo.Agent = s // 設置代理為 Nats Server
			s.server.Call(*callInfo)
		} else {
			log.Error("Unmarshal error:", err)
		}
	}

	return nil
}

// 保存解碼後的數據，Value可以為任意數據類型
func (s *NatsServer) Unmarshal(data []byte) (*rpcPB.RPCInfo, error) {
	var rpcInfo rpcPB.RPCInfo
	err := proto.Unmarshal(data, &rpcInfo)
	if err != nil {
		return nil, err
	} else {
		return &rpcInfo, err
	}
}

// goroutine safe
func (s *NatsServer) MarshalResult(resultInfo rpcPB.ResultInfo) ([]byte, error) {
	b, err := proto.Marshal(&resultInfo)
	return b, err
}
