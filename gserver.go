package gserver

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"

	_ "github.com/joho/godotenv/autoload"
	module "github.com/shihray/gserver/module"
)

const version = "1"

// 創建模組
func CreateApp(opts ...module.Option) module.App {
	opts = append(opts, module.Version(version))
	// return app.NewApp(opts...)
	return nil // 實作完成修改回來
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
