package defaultrpc

import rpcpb "github.com/shihray/gserver/rpc/pb"

type ClinetCallInfo struct {
	correlationID string // 關聯ID
	timeout       int64  // 超時
	call          chan rpcpb.ResultInfo
}
