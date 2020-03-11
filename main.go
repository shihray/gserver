package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"

	_ "github.com/joho/godotenv/autoload"
)

const version = "v1"

func init() {

}

func main() {
	// 開啟效能分析
	go func() {
		ip := "0.0.0.0:6060"
		if err := http.ListenAndServe(ip, nil); err != nil {
			fmt.Printf("start pprof failed on %s\n", ip)
		}
	}()
}
