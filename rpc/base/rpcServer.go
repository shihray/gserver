package defaultrpc

import (
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	logging "github.com/shihray/gserver/logging"
	module "github.com/shihray/gserver/module"
	mqrpc "github.com/shihray/gserver/rpc"
	rpcpb "github.com/shihray/gserver/rpc/pb"
	argsutil "github.com/shihray/gserver/rpc/util"
	utils "github.com/shihray/gserver/utils"
)

type RPCServer struct {
	module       module.Module
	app          module.App
	functions    map[string]*mqrpc.FunctionInfo
	natsServer   *NatsServer
	mq_chan      chan mqrpc.CallInfo //接收到請求信息的隊列
	wg           sync.WaitGroup      //任務阻塞
	callChanDone chan error
	listener     mqrpc.RPCListener
	control      mqrpc.GoroutineControl //控制模塊可同時開啟的最大協程數
	executing    int64                  //正在執行的goroutine數量
}

func NewRPCServer(app module.App, module module.Module) (mqrpc.RPCServer, error) {
	rpcServer := new(RPCServer)
	rpcServer.app = app
	rpcServer.module = module
	rpcServer.callChanDone = make(chan error)
	rpcServer.functions = make(map[string]*mqrpc.FunctionInfo)
	rpcServer.mq_chan = make(chan mqrpc.CallInfo)

	natsServer, err := NewNatsServer(app, rpcServer)
	if err != nil {
		logging.Error("AMQPServer Dial: %s", err)
	}
	rpcServer.natsServer = natsServer

	return rpcServer, nil
}

func (this *RPCServer) Addr() string {
	return this.natsServer.Addr()
}

func (s *RPCServer) SetListener(listener mqrpc.RPCListener) {
	s.listener = listener
}

func (s *RPCServer) SetGoroutineControl(control mqrpc.GoroutineControl) {
	s.control = control
}

/**
獲取當前正在執行的goroutine 數量
*/
func (s *RPCServer) GetExecuting() int64 {
	return s.executing
}

// you must call the function before calling Open and Go
func (s *RPCServer) Register(id string, f interface{}) {

	if _, ok := s.functions[id]; ok {
		panic(fmt.Sprintf("function id %v: already registered", id))
	}

	s.functions[id] = &mqrpc.FunctionInfo{
		Function:  reflect.ValueOf(f),
		Goroutine: false,
	}
}

// you must call the function before calling Open and Go
func (s *RPCServer) RegisterGO(id string, f interface{}) {

	if _, ok := s.functions[id]; ok {
		panic(fmt.Sprintf("function id %v: already registered", id))
	}

	s.functions[id] = &mqrpc.FunctionInfo{
		Function:  reflect.ValueOf(f),
		Goroutine: true,
	}
}

func (s *RPCServer) Done() (err error) {
	//等待正在執行的請求完成
	s.wg.Wait()
	//關閉隊列鏈接
	if s.natsServer != nil {
		err = s.natsServer.Shutdown()
	}
	return
}

func (s *RPCServer) Call(callInfo mqrpc.CallInfo) error {
	s.runFunc(callInfo)
	return nil
}

/**
接收請求信息
*/
func (s *RPCServer) on_call_handle(calls <-chan mqrpc.CallInfo, done chan error) {
	for {
		select {
		case callInfo, ok := <-calls:
			if !ok {
				goto ForEnd
			} else {
				if callInfo.RpcInfo.Expired < (time.Now().UnixNano() / utils.Nano2Millisecond) {
					//請求超時了,無需再處理
					if s.listener != nil {
						s.listener.OnTimeOut(callInfo.RpcInfo.Fn, callInfo.RpcInfo.Expired)
					} else {
						logging.Warn("timeout: This is Call", s.module.GetType(), callInfo.RpcInfo.Fn, callInfo.RpcInfo.Expired, time.Now().UnixNano()/utils.Nano2Millisecond)
					}
				} else {
					s.runFunc(callInfo)
				}
			}
		case <-done:
			goto ForEnd
		}
	}
ForEnd:
}

func (s *RPCServer) doCallback(callInfo mqrpc.CallInfo) {
	if callInfo.RpcInfo.Reply {
		// 需要回覆的才回覆
		err := callInfo.Agent.(mqrpc.MQServer).Callback(callInfo)
		if err != nil {
			logging.Warn("rpc callback erro :\n%s", err.Error())
		}
	} else {
		// 對於不需要回覆的消息,可以判斷一下是否出現錯誤，打印一些警告
		if callInfo.Result.Error != "" {
			logging.Warn("rpc callback erro :\n%s", callInfo.Result.Error)
		}
	}
	if s.app.Options().ServerRPCHandler != nil {
		s.app.Options().ServerRPCHandler(s.app, s.module, callInfo)
	}
}

// if _func is not a function or para num and type not match,it will cause panic
func (s *RPCServer) runFunc(callInfo mqrpc.CallInfo) {
	start := time.Now()
	iErrorCallback := func(Cid string, Error string) {
		// 異常日志都應該打印
		resultInfo := rpcpb.NewResultInfo(Cid, Error, argsutil.NULL, nil)
		callInfo.Result = *resultInfo
		callInfo.ExecTime = time.Since(start).Nanoseconds()
		s.doCallback(callInfo)
		if s.listener != nil {
			s.listener.OnError(callInfo.RpcInfo.Fn, &callInfo, fmt.Errorf(Error))
		}
	}
	defer utils.RecoverFunc()

	functionInfo, ok := s.functions[callInfo.RpcInfo.Fn]
	if !ok {
		if s.listener != nil {
			fInfo, err := s.listener.NoFoundFunction(callInfo.RpcInfo.Fn)
			if err != nil {
				iErrorCallback(callInfo.RpcInfo.Cid, err.Error())
				return
			}
			functionInfo = fInfo
		}
	}
	f := functionInfo.Function
	params := callInfo.RpcInfo.Args
	ArgsType := callInfo.RpcInfo.ArgsType
	if len(params) != f.Type().NumIn() {
		//因為在調研的 _func的時候還會額外傳遞一個回調函數 cb
		iErrorCallback(callInfo.RpcInfo.Cid, fmt.Sprintf("The number of params %v is not adapted.%v", params, f.String()))
		return
	}

	iRunFunc := func() {
		s.wg.Add(1)
		s.executing++
		defer func() {
			s.wg.Add(-1)
			s.executing--
			if s.control != nil {
				s.control.Finish()
			}
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
				allError := fmt.Sprintf("%s rpc func(%s) error %s\n ----Stack----\n%s", s.module.GetType(), callInfo.RpcInfo.Fn, rn, errstr)
				logging.Error(allError)
				iErrorCallback(callInfo.RpcInfo.Cid, allError)
			}
		}()

		var in []reflect.Value
		if len(ArgsType) < 1 {
			return
		}
		in = make([]reflect.Value, len(params))
		for k, v := range ArgsType {
			rv := f.Type().In(k)
			var elemp reflect.Value
			if rv.Kind() == reflect.Ptr {
				elemp = reflect.New(rv.Elem())
			} else {
				elemp = reflect.New(rv)
			}

			if pb, ok := elemp.Interface().(mqrpc.Marshaler); ok {
				err := pb.Unmarshal(params[k])
				if err != nil {
					iErrorCallback(callInfo.RpcInfo.Cid, err.Error())
					return
				}
				if rv.Kind() == reflect.Ptr {
					//接收指針變量的參數
					in[k] = reflect.ValueOf(elemp.Interface())
				} else {
					//接收值變量
					in[k] = elemp.Elem()
				}
			} else if pb, ok := elemp.Interface().(proto.Message); ok {
				err := proto.Unmarshal(params[k], pb)
				if err != nil {
					iErrorCallback(callInfo.RpcInfo.Cid, err.Error())
					return
				}
				if rv.Kind() == reflect.Ptr {
					//接收指針變量的參數
					in[k] = reflect.ValueOf(elemp.Interface())
				} else {
					//接收值變量
					in[k] = elemp.Elem()
				}
			} else {
				//不是Marshaler 才嘗試用 argsutil 解析
				ty, err := argsutil.Bytes2Args(s.app, v, params[k])
				if err != nil {
					iErrorCallback(callInfo.RpcInfo.Cid, err.Error())
					return
				}
				switch v2 := ty.(type) {
				case nil:
					in[k] = reflect.Zero(f.Type().In(k))
				case []uint8:
					if reflect.TypeOf(ty).AssignableTo(f.Type().In(k)) {
						//如果ty "繼承" 於接受參數類型
						in[k] = reflect.ValueOf(ty)
					} else {
						elemp := reflect.New(f.Type().In(k))
						err := json.Unmarshal(v2, elemp.Interface())
						if err != nil {
							logging.Error("%v []uint8--> %v error with='%v'", callInfo.RpcInfo.Fn, f.Type().In(k), err)
							in[k] = reflect.ValueOf(ty)
						} else {
							in[k] = elemp.Elem()
						}
					}
				default:
					in[k] = reflect.ValueOf(ty)
				}
			}
		}

		if s.listener != nil {
			errs := s.listener.BeforeHandle(callInfo.RpcInfo.Fn, &callInfo)
			if errs != nil {
				iErrorCallback(callInfo.RpcInfo.Cid, errs.Error())
				return
			}
		}

		out := f.Call(in)
		var rs []interface{}
		if len(out) != 2 {
			iErrorCallback(callInfo.RpcInfo.Cid, fmt.Sprintf("%s rpc func(%s) return error %s\n", s.module.GetType(), callInfo.RpcInfo.Fn, "func(....)(result interface{}, err error)"))
			return
		}
		// prepare out paras
		if len(out) > 0 {
			rs = make([]interface{}, len(out), len(out))
			for i, v := range out {
				rs[i] = v.Interface()
			}
		}
		var rerr string
		switch e := rs[1].(type) {
		case string:
			rerr = e
			break
		case error:
			rerr = e.Error()
		case nil:
			rerr = ""
		default:
			iErrorCallback(callInfo.RpcInfo.Cid, fmt.Sprintf("%s rpc func(%s) return error %s\n", s.module.GetType(), callInfo.RpcInfo.Fn, "func(....)(result interface{}, err error)"))
			return
		}
		argsType, args, err := argsutil.ArgsTypeAnd2Bytes(s.app, rs[0])
		if err != nil {
			iErrorCallback(callInfo.RpcInfo.Cid, err.Error())
			return
		}
		resultInfo := rpcpb.NewResultInfo(
			callInfo.RpcInfo.Cid,
			rerr,
			argsType,
			args,
		)
		callInfo.Result = *resultInfo
		callInfo.ExecTime = time.Since(start).Nanoseconds()
		s.doCallback(callInfo)
		logging.Debug("RPC Exec ModuleType = %v Func = %v Elapsed = %v", s.module.GetType(), callInfo.RpcInfo.Fn, time.Since(start))
		if s.listener != nil {
			s.listener.OnComplete(callInfo.RpcInfo.Fn, &callInfo, resultInfo, time.Since(start).Nanoseconds())
		}
	}

	// 協程數量達到最大限制
	if s.control != nil {
		s.control.Wait()
	}
	if functionInfo.Goroutine {
		go iRunFunc()
	} else {
		iRunFunc()
	}
}
