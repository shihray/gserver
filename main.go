package main

import (
	"github.com/nats-io/nats.go"
	ping "github.com/shihray/gserver/demoPING"
	pong "github.com/shihray/gserver/demoPONG"
	ModuleRegistry "github.com/shihray/gserver/registry"
	CommonConf "github.com/shihray/gserver/utils/conf"
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
	natsUrl := "nats://127.0.0.1:14222,nats://127.0.0.1:16222,nats://127.0.0.1:18222"
	//natsUrl := CommonConf.GetEnv("NatsURL", nats.DefaultURL)

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

	var registryOption Module.Option
	registersUrl := CommonConf.GetEnv("Registers_Url", "")
	if registersUrl != "" {
		// consol註冊
		rsConsul := ModuleRegistry.NewConsulRegistry(func(op *ModuleRegistry.Options) {
			op.Addrs = []string{
				registersUrl,
			}
		})
		registryOption = Module.Registry(rsConsul)
	} else {
		// redis ModuleRegistry 註冊
		rsRedis := ModuleRegistry.NewRedisRegistry(func(op *ModuleRegistry.Options) {
			op.RedisHost = "localhost:6379"
			op.RedisPassword = ""
		})
		registryOption = Module.Registry(rsRedis)
	}
	//log.Println(registryOption)

	app := CreateApp(
		Module.Version(version),  // version
		Module.Nats(nc),          // nats
		registryOption,           // register
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
