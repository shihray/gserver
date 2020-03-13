// 此功能預計加入與client端溝通之 websocket連線
// 預計將整合 `goserverframework` websocket server & client.
// 目前此功能尚未開發，開發預計會加入 gate module
package network

import (
	"net"
)

type Agent interface {
	Run() error
	OnClose() error
}

type Conn interface {
	net.Conn
	ConnHeader() map[string]interface{}
	Destroy()
	doDestroy()
}
