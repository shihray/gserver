package main

import (
	"fmt"
	"github.com/nats-io/nats.go"
	"net/http"
	_ "net/http/pprof"

	_ "github.com/joho/godotenv/autoload"

	ping "github.com/shihray/gserver/demoPING"
	pong "github.com/shihray/gserver/demoPONG"
	Module "github.com/shihray/gserver/module"
	Conf "github.com/shihray/gserver/utils/conf"

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
	natsUrl := Conf.GetEnv("NatsURL", nats.DefaultURL)
	nc, err := nats.Connect(natsUrl, nats.MaxReconnects(10000))
	if err != nil {
		fmt.Println("Nats Connect Error ", err.Error())
		return
	}
	fmt.Println("Connect to Nats Server... ", nc.ConnectedAddr())

	app := CreateApp(
		Module.Version(version), // version
		Module.Nats(nc),         // nats register
	)

	erro := app.Run(
		ping.Module(),
		pong.Module(),
	)
	if erro != nil {
		fmt.Println("App Work[Run] Error", erro.Error())
		return
	}
	fmt.Println("Start Server...", app)
}
