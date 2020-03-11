package defaultrpc

import rpcpb "github.com/shihray/gserver/modules/rpc/pb"

type ClinetCallInfo struct {
	correlation_id string
	timeout        int64 //超时
	call           chan rpcpb.ResultInfo
}
