package main

import (
	"github.com/nats-io/nats.go"
	ping "github.com/shihray/gserver/demoPING"
	pong "github.com/shihray/gserver/demoPONG"
	"github.com/shihray/gserver/registry"
	"github.com/shihray/gserver/utils/conf"
	"log"
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
			log.Printf("start pprof failed on %s\n", ip)
		}
	}()
}

func main() {
	ListenServe()
	// nats setting
	// connect to multi servers
	natsUrl := "nats://127.0.0.1:4222,nats://127.0.0.1:5222,nats://127.0.0.1:6222"
	//natsUrl := Conf.GetEnv("NatsURL", nats.DefaultURL)

	var opts = []nats.Option{
		nats.DontRandomize(), // turn off randomizing the server pool.
		nats.MaxReconnects(10000),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("Got disconnected! Reason: %q\n", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("Got reconnected to %v!\n", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Printf("Connection closed. Reason: %q\n", nc.LastError())
		}),
	}
	nc, err := nats.Connect(natsUrl, opts...)
	if err != nil {
		log.Println("Nats Connect Error ", err.Error())
		return
	}
	log.Println("Connect to Nats Server... ", nc.ConnectedAddr())

	// consol註冊
	registersUrl := conf.GetEnv("Registers_Url", "127.0.0.1:8500")
	rs := registry.NewConsulRegistry(func(op *registry.Options) {
		op.Addrs = []string{
			registersUrl,
		}
	})

	app := CreateApp(
		Module.Version(version),  // version
		Module.Nats(nc),          // nats register
		Module.Registry(rs),      // consul register
		Module.RoutineCount(100), // routine size
	)

	erro := app.Run(
		ping.Module(),
		pong.Module(),
	)
	if erro != nil {
		log.Println("App Work[Run] Error", erro.Error())
		return
	}
}
