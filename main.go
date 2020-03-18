package main

import (
	"fmt"
	"github.com/nats-io/nats.go"
	ping "github.com/shihray/gserver/demoPING"
	pong "github.com/shihray/gserver/demoPONG"
	"net/http"
	_ "net/http/pprof"

	_ "github.com/joho/godotenv/autoload"

	Module "github.com/shihray/gserver/module"
	moduleUtil "github.com/shihray/gserver/source/moduleutil"
)

const version = "1"

// 創建模組
func CreateApp(opts ...Module.Option) Module.App {
	return moduleUtil.NewApp(opts...)
}

// 開啟效能分析
func ListenServe() {
	go func() {
		ip := "0.0.0.0:6060"
		if err := http.ListenAndServe(ip, nil); err != nil {
			fmt.Printf("start pprof failed on %s\n", ip)
		}
	}()
}

func main() {
	ListenServe()
	// nats setting
	// connect to multi servers
	natsUrl := "nats://127.0.0.1:4222,nats://127.0.0.1:5222,nats://127.0.0.1:6222"
	//natsUrl := Conf.GetEnv("NatsURL", nats.DefaultURL)
	nc, err := nats.Connect(natsUrl,
		nats.MaxReconnects(10000),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			fmt.Printf("Got disconnected! Reason: %q\n", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Printf("Got reconnected to %v!\n", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			fmt.Printf("Connection closed. Reason: %q\n", nc.LastError())
		}),
	)
	if err != nil {
		fmt.Println("Nats Connect Error ", err.Error())
		return
	}
	fmt.Println("Connect to Nats Server... ", nc.ConnectedAddr())

	app := CreateApp(
		Module.Version(version),  // version
		Module.Nats(nc),          // nats register
		Module.RoutineCount(100), // routine size
	)

	erro := app.Run(
		ping.Module(),
		pong.Module(),
	)
	if erro != nil {
		fmt.Println("App Work[Run] Error", erro.Error())
		return
	}
}
