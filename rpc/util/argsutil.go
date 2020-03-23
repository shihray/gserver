package argsutil

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
	logging "github.com/shihray/gserver/logging"
	module "github.com/shihray/gserver/module"
	mqrpc "github.com/shihray/gserver/rpc"
	"github.com/shihray/gserver/utils"
)

var (
	NULL    = "null"    //nil null
	BOOL    = "bool"    //bool
	INT     = "int"     //int
	LONG    = "long"    //long64
	FLOAT   = "float"   //float32
	DOUBLE  = "double"  //float64
	BYTES   = "bytes"   //[]byte
	STRING  = "string"  //string
	MAP     = "map"     //map[string]interface{}
	MAPSTR  = "mapstr"  //map[string]string{}
	Marshal = "marshal" //mqrpc.Marshaler
	Proto   = "proto"   //proto.Message
)

func ArgsTypeAnd2Bytes(app module.App, arg interface{}) (string, []byte, error) {
	if arg == nil {
		return NULL, nil, nil
	}
	switch v2 := arg.(type) {
	case []uint8:
		return BYTES, v2, nil
	}
	switch v2 := arg.(type) {
	case nil:
		return NULL, nil, nil
	case string:
		return STRING, []byte(v2), nil
	case bool:
		return BOOL, utils.BoolToBytes(v2), nil
	case int32:
		return INT, utils.Int32ToBytes(v2), nil
	case int64:
		return LONG, utils.Int64ToBytes(v2), nil
	case float32:
		return FLOAT, utils.Float32ToBytes(v2), nil
	case float64:
		return DOUBLE, utils.Float64ToBytes(v2), nil
	case []byte:
		return BYTES, v2, nil
	case map[string]interface{}:
		bytes, err := utils.MapToBytes(v2)
		if err != nil {
			return MAP, nil, err
		}
		return MAP, bytes, nil
	case map[string]string:
		bytes, err := utils.MapToBytesString(v2)
		if err != nil {
			return MAPSTR, nil, err
		}
		return MAPSTR, bytes, nil
	default:
		for _, v := range app.GetRPCSerialize() {
			ptype, vk, err := v.Serialize(arg)
			if err == nil {
				//解析成功了
				return ptype, vk, err
			}
		}

		rv := reflect.ValueOf(arg)
		//不是指针
		if rv.Kind() != reflect.Ptr {
			return "", nil, fmt.Errorf("Args2Bytes [%v] not registered to app.addrpcserialize(...) structure type or not *mqrpc.marshaler pointer type", reflect.TypeOf(arg))
		} else {
			if rv.IsNil() {
				//如果是nil则直接返回
				return NULL, nil, nil
			}

			if b, err := jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(arg); err != nil {
				return "", nil, fmt.Errorf("args [%s] marshal error %v", reflect.TypeOf(arg), err)
			} else {
				return Marshal, b, nil
			}

			if v2, ok := arg.(mqrpc.Marshaler); ok {
				b, err := v2.Marshal()
				if err != nil {
					return "", nil, fmt.Errorf("args [%s] marshal error %v", reflect.TypeOf(arg), err)
				}
				if v2.String() != "" {
					return fmt.Sprintf("%v@%v", Marshal, v2.String()), b, nil
				} else {
					return fmt.Sprintf("%v@%v", Marshal, reflect.TypeOf(arg)), b, nil
				}
			}
			if v2, ok := arg.(proto.Message); ok {
				b, err := proto.Marshal(v2)
				if err != nil {
					logging.Error("proto.Marshal error")
					return "", nil, fmt.Errorf("args [%s] proto.Marshal error %v", reflect.TypeOf(arg), err)
				}
				if v2.String() != "" {
					return fmt.Sprintf("%v@%v", Proto, v2.String()), b, nil
				} else {
					return fmt.Sprintf("%v@%v", Proto, reflect.TypeOf(arg)), b, nil
				}
			}
		}

		return "", nil, fmt.Errorf("Args2Bytes [%s] not registered to app.addrpcserialize(...) structure type", reflect.TypeOf(arg))
	}
}

func Bytes2Args(app module.App, argsType string, args []byte) (interface{}, error) {
	if strings.HasPrefix(argsType, Marshal) {
		return args, nil
	}
	if strings.HasPrefix(argsType, Proto) {
		return args, nil
	}
	switch argsType {
	case NULL:
		return nil, nil
	case STRING:
		return string(args), nil
	case BOOL:
		return utils.BytesToBool(args), nil
	case INT:
		return utils.BytesToInt32(args), nil
	case LONG:
		return utils.BytesToInt64(args), nil
	case FLOAT:
		return utils.BytesToFloat32(args), nil
	case DOUBLE:
		return utils.BytesToFloat64(args), nil
	case BYTES:
		return args, nil
	case MAP:
		mps, errs := utils.BytesToMap(args)
		if errs != nil {
			return nil, errs
		}
		return mps, nil
	case MAPSTR:
		mps, errs := utils.BytesToMapString(args)
		if errs != nil {
			return nil, errs
		}
		return mps, nil
	default:
		for _, v := range app.GetRPCSerialize() {
			vk, err := v.Deserialize(argsType, args)
			if err == nil {
				//解析成功了
				return vk, err
			}
		}
		return nil, fmt.Errorf("Bytes2Args [%s] not registered to app.addrpcserialize(...)", argsType)
	}
}
