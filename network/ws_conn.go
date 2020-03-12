package network

import (
	"bytes"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WebsocketConnSet map[*websocket.Conn]struct{}

type WSConn struct {
	io.Reader //Read(p []byte) (n int, err error)
	io.Writer //Write(p []byte) (n int, err error)
	sync.Mutex
	buf_lock  chan error //当有写入一次数据设置一次
	buffer    bytes.Buffer
	conn      *websocket.Conn
	readfirst bool
	closeFlag bool
}

func newWSConn(conn *websocket.Conn) *WSConn {
	wsConn := new(WSConn)
	wsConn.conn = conn
	wsConn.buf_lock = make(chan error)
	wsConn.readfirst = false
	go func() {
		for {
			_, b, err := wsConn.conn.ReadMessage()
			if err != nil {
				//logging.Error("读取数据失败 %s",err.Error())
				wsConn.buf_lock <- err
				break
			} else {
				wsConn.buffer.Write(b)
				wsConn.readfirst = true
				wsConn.buf_lock <- nil
			}
		}
		conn.Close()
		wsConn.Lock()
		wsConn.closeFlag = true
		wsConn.Unlock()
	}()

	return wsConn
}

func (wsConn *WSConn) doDestroy() {
	wsConn.conn.UnderlyingConn().(*net.TCPConn).SetLinger(0)
	wsConn.conn.Close()

	if !wsConn.closeFlag {
		wsConn.closeFlag = true
	}
}

func (wsConn *WSConn) Destroy() {
	wsConn.Lock()
	defer wsConn.Unlock()

	wsConn.doDestroy()
}

func (wsConn *WSConn) Close() error {
	wsConn.Lock()
	defer wsConn.Unlock()
	if wsConn.closeFlag {
		return nil
	}
	wsConn.closeFlag = true
	return wsConn.conn.Close()
}

func (wsConn *WSConn) Write(p []byte) (int, error) {
	err := wsConn.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// goroutine not safe
func (wsConn *WSConn) Read(p []byte) (n int, err error) {
	err = <-wsConn.buf_lock //等待写入数据
	if err != nil {
		//读取数据出现异常了
		return
	}
	if wsConn.buffer.Len() == 0 {
		//再等一次
		err = <-wsConn.buf_lock //等待写入数据
		if err != nil {
			//读取数据出现异常了
			return
		}
	}
	return wsConn.buffer.Read(p)
}

func (wsConn *WSConn) LocalAddr() net.Addr {
	return wsConn.conn.LocalAddr()
}

func (wsConn *WSConn) RemoteAddr() net.Addr {
	return wsConn.conn.RemoteAddr()
}

// A zero value for t means I/O operations will not time out.
func (wsConn *WSConn) SetDeadline(t time.Time) error {
	err := wsConn.conn.SetWriteDeadline(t)
	if err != nil {
		return err
	}
	return wsConn.conn.SetWriteDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls.
// A zero value for t means Read will not time out.
func (wsConn *WSConn) SetReadDeadline(t time.Time) error {
	return wsConn.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (wsConn *WSConn) SetWriteDeadline(t time.Time) error {
	return wsConn.conn.SetWriteDeadline(t)
}
