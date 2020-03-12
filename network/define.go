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
