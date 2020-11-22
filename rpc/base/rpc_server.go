package defaultrpc

import (
	"fmt"
	jsonIter "github.com/json-iterator/go"
	"reflect"
	"runtime"
	"sync"
	"time"

	module "github.com/shihray/gserver/module"
	mqRPC "github.com/shihray/gserver/rpc"
	rpcPB "github.com/shihray/gserver/rpc/pb"
	argsUtil "github.com/shihray/gserver/rpc/util"
	utils "github.com/shihray/gserver/utils"
	log "github.com/z9905080/gloger"
)

type RPCServer struct {
	module       module.Module
	app          module.App
	functions    map[string]*mqRPC.FunctionInfo
	natsServer   *NatsServer
	mqChan       chan mqRPC.CallInfo // 接收到請求信息的隊列
	wg           sync.WaitGroup      // 任務阻塞
	callChanDone chan error
	listener     mqRPC.RPCListener
	control      mqRPC.GoroutineControl // 控制模塊可同時開啟的最大協程數
	executing    int64                  // 正在執行的goroutine數量
}

func NewRPCServer(app module.App, module module.Module) (mqRPC.RPCServer, error) {
	rpcServer := new(RPCServer)
	rpcServer.app = app
	rpcServer.module = module
	rpcServer.callChanDone = make(chan error)
	rpcServer.functions = make(map[string]*mqRPC.FunctionInfo)
	rpcServer.mqChan = make(chan mqRPC.CallInfo)

	natsServer, err := NewNatsServer(app, rpcServer)
	if err != nil {
		log.Error("Nats RPC Server Create Dial Error: ", err)
	}
	rpcServer.natsServer = natsServer

	return rpcServer, nil
}

func (s *RPCServer) Addr() string {
	return s.natsServer.Addr()
}

func (s *RPCServer) SetListener(listener mqRPC.RPCListener) {
	s.listener = listener
}

func (s *RPCServer) SetGoroutineControl(control mqRPC.GoroutineControl) {
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

	s.functions[id] = &mqRPC.FunctionInfo{
		Function:  reflect.ValueOf(f),
		Goroutine: false,
	}
}

// you must call the function before calling Open and Go
func (s *RPCServer) RegisterGO(id string, f interface{}) {

	if _, ok := s.functions[id]; ok {
		panic(fmt.Sprintf("function id %v: already registered", id))
	}

	s.functions[id] = &mqRPC.FunctionInfo{
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

func (s *RPCServer) Call(callInfo mqRPC.CallInfo) error {
	s.runFunc(callInfo)
	return nil
}

func (s *RPCServer) doCallback(callInfo mqRPC.CallInfo) {
	if callInfo.RpcInfo.Reply {
		// 需要回覆的才回覆
		err := callInfo.Agent.(mqRPC.MQServer).Callback(callInfo)
		if err != nil {
			log.WarnF("rpc callback erro :\n%s", err.Error())
		}
	} else {
		if callInfo.Result.Error != "" {
			log.WarnF("rpc callback erro :\n%s", callInfo.Result.Error)
		}
	}
	if s.app.Options().ServerRPCHandler != nil {
		s.app.Options().ServerRPCHandler(s.app, s.module, callInfo)
	}
}

// if _func is not a function or para num and type not match,it will cause panic
func (s *RPCServer) runFunc(callInfo mqRPC.CallInfo) {
	start := time.Now()
	iErrorCallback := func(Cid string, Error string) {
		// print all error logs
		resultInfo := rpcPB.NewResultInfo(Cid, Error, argsUtil.NULL, nil)
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
			s.wg.Done()
			s.executing--
			if s.control != nil {
				s.control.Finish(1)
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
				log.Error(allError)
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
			var elem reflect.Value
			if rv.Kind() == reflect.Ptr {
				elem = reflect.New(rv.Elem())
			} else {
				elem = reflect.New(rv)
			}

			jsonToolElem := reflect.New(f.Type().In(k))
			if err := jsonIter.ConfigCompatibleWithStandardLibrary.Unmarshal(params[k], jsonToolElem.Interface()); err == nil {
				in[k] = jsonToolElem.Elem()
			}

			if pb, ok := elem.Interface().(mqRPC.Marshaler); ok {
				err := pb.Unmarshal(params[k])
				if err != nil {
					iErrorCallback(callInfo.RpcInfo.Cid, err.Error())
					return
				}
				if rv.Kind() == reflect.Ptr {
					//接收指針變量的參數
					in[k] = reflect.ValueOf(elem.Interface())
				} else {
					//接收值變量
					in[k] = elem.Elem()
				}
			} else if _, ok := elem.Interface().([]byte); ok {
				in[k] = elem.Elem()
			} else {
				// 不是Marshaler 才嘗試用 argsUtil 解析
				ty, err := argsUtil.Bytes2Args(s.app, v, params[k])
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
						elem := reflect.New(f.Type().In(k))
						err := jsonIter.ConfigCompatibleWithStandardLibrary.Unmarshal(v2, elem.Interface())
						if err != nil {
							log.ErrorF("%v []uint8--> %v error with='%v'", callInfo.RpcInfo.Fn, f.Type().In(k), err)
							in[k] = reflect.ValueOf(ty)
						} else {
							in[k] = elem.Elem()
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
			rs = make([]interface{}, len(out))
			for i, v := range out {
				rs[i] = v.Interface()
			}
		}
		var rErr string
		switch e := rs[1].(type) {
		case string:
			rErr = e
		case error:
			rErr = e.Error()
		case nil:
			rErr = ""
		default:
			iErrorCallback(
				callInfo.RpcInfo.Cid,
				fmt.Sprintf("%s rpc func(%s) return error %s\n", s.module.GetType(), callInfo.RpcInfo.Fn, "func(....)(result interface{}, err error)"),
			)
			return
		}
		argsType, args, err := argsUtil.ArgsTypeAnd2Bytes(s.app, rs[0])
		if err != nil {
			iErrorCallback(callInfo.RpcInfo.Cid, err.Error())
			return
		}
		resultInfo := rpcPB.NewResultInfo(
			callInfo.RpcInfo.Cid,
			rErr,
			argsType,
			args,
		)
		callInfo.Result = *resultInfo
		callInfo.ExecTime = time.Since(start).Nanoseconds()
		s.doCallback(callInfo)
		msg := fmt.Sprintf("RPC Exec ModuleType = %v Func = %v Elapsed = %v", s.module.GetType(), callInfo.RpcInfo.Fn, time.Since(start))
		log.Debug(msg)
		if s.listener != nil {
			s.listener.OnComplete(callInfo.RpcInfo.Fn, &callInfo, resultInfo, time.Since(start).Nanoseconds())
		}
	}

	if s.control != nil {
		s.control.Start(1)
	}

	if functionInfo.Goroutine {
		go iRunFunc()
	} else {
		iRunFunc()
	}
}
