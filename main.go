package main

import (
	"fmt"
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
}
