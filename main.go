package main

import (
	"github.com/nats-io/nats.go"
	ping "github.com/shihray/gserver/demoPING"
	pong "github.com/shihray/gserver/demoPONG"
	Module "github.com/shihray/gserver/module"
	ModuleRegistry "github.com/shihray/gserver/registry"
	moduleUtil "github.com/shihray/gserver/source/moduleutil"
	CommonConf "github.com/shihray/gserver/utils/conf"
	log "github.com/z9905080/gloger"
	"net/http"
	_ "net/http/pprof"
	"time"
)

const version = "0.0.14"

// 創建模組
func CreateApp(opts ...Module.Option) Module.App {
	return moduleUtil.NewApp(opts...)
}

// 開啟效能分析
func ListenServe() {
	go func() {
		ip := "0.0.0.0:6060"
		if err := http.ListenAndServe(ip, nil); err != nil {
			log.ErrorF("start pprof failed on %s", ip)
		}
	}()
}

func main() {
	ListenServe()
	// nats setting
	// connect to multi servers
	//natsUrl := "nats://127.0.0.1:14222,nats://127.0.0.1:16222,nats://127.0.0.1:18222"
	natsUrl := CommonConf.GetEnv("NatsURL", nats.DefaultURL)
	registersUrl := CommonConf.GetEnv("Registers_Url", "")
	var opts = []nats.Option{
		nats.DontRandomize(), // turn off randomizing the server pool.
		nats.MaxReconnects(10000),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.ErrorF("Got disconnected! Reason: %q", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.ErrorF("Got reconnected to %v", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.ErrorF("Connection closed. Reason: %q", nc.LastError())
		}),
	}
	nc, err := nats.Connect(natsUrl, opts...)
	if err != nil {
		log.Error("Nats Connect Error ", err.Error())
		return
	}
	log.Info("Connect to Nats Server... ", nc.ConnectedAddr())

	var registryOption Module.Option
	if registersUrl != "" {
		// consul註冊
		rsConsul := ModuleRegistry.NewConsulRegistry(func(op *ModuleRegistry.Options) {
			op.Addrs = []string{
				registersUrl,
			}
		})
		registryOption = Module.Registry(rsConsul)
	} else {
		// redis ModuleRegistry 註冊
		rsRedis := ModuleRegistry.NewRedisRegistry(func(op *ModuleRegistry.Options) {
			op.RedisHost = "127.0.0.1:6379"
			op.RedisPassword = ""
			op.GroupID = 2
		})
		registryOption = Module.Registry(rsRedis)
	}

	routineCount := 100
	app := CreateApp(
		Module.LogMode(int(log.Stdout)),        // log mode 0:Stdout 1:file
		Module.LogLevel(int(log.DEBUG)),        // 0:debug 1:Info 2:Warn 3:Error 4:Fatal
		Module.Version(version),                // version
		Module.Nats(nc),                        // nats
		registryOption,                         // register
		Module.RegisterInterval(1*time.Second), // RegisterInterval
		Module.RoutineCount(routineCount),      // routine size
	)
	// init modules
	erro := app.Run(
		ping.Module(),
		pong.Module(),
		pong.Module(),
	)
	if erro != nil {
		log.Error("App Work[Run] Error", erro.Error())
		return
	}
}
