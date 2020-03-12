package defaultrpc

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats.go"
	logging "github.com/shihray/gserver/logging"
	module "github.com/shihray/gserver/module"
	mqrpc "github.com/shihray/gserver/rpc"
	rpcpb "github.com/shihray/gserver/rpc/pb"
	"github.com/shihray/gserver/utils"
)

type NatsServer struct {
	callChan chan mqrpc.CallInfo
	addr     string
	app      module.App
	server   *RPCServer
	done     chan error
	isClose  bool
}

func setAddrs(addrs []string) []string {
	var cAddrs []string
	for _, addr := range addrs {
		if len(addr) == 0 {
			continue
		}
		if !strings.HasPrefix(addr, "nats://") {
			addr = "nats://" + addr
		}
		cAddrs = append(cAddrs, addr)
	}
	if len(cAddrs) == 0 {
		cAddrs = []string{nats.DefaultURL}
	}
	return cAddrs
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
	reply_to := callinfo.Props["reply_to"].(string)
	return s.app.Transport().Publish(reply_to, body)
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
			logging.Error("NatsServer error with '%v'", err)
			continue
		}

		rpcInfo, err := s.Unmarshal(m.Data)
		if err == nil {
			callInfo := &mqrpc.CallInfo{
				RpcInfo: *rpcInfo,
			}
			callInfo.Props = map[string]interface{}{
				"reply_to": rpcInfo.ReplyTo,
			}
			callInfo.Agent = s //設置代理為NatsServer
			s.server.Call(*callInfo)
		} else {
			fmt.Println("error ", err)
		}
	}

	return nil
}

// 保存解碼後的數據，Value可以為任意數據類型
func (s *NatsServer) Unmarshal(data []byte) (*rpcpb.RPCInfo, error) {
	var rpcInfo rpcpb.RPCInfo
	err := proto.Unmarshal(data, &rpcInfo)
	if err != nil {
		return nil, err
	} else {
		return &rpcInfo, err
	}
}

// goroutine safe
func (s *NatsServer) MarshalResult(resultInfo rpcpb.ResultInfo) ([]byte, error) {
	b, err := proto.Marshal(&resultInfo)
	return b, err
}
