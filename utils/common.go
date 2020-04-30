package utils

import (
	"fmt"
	"log"
	"runtime"

	logging "github.com/shihray/gserver/logging"
	"github.com/shihray/gserver/utils/conf"
)

const Nano2Millisecond int64 = 1000000

func RecoverFunc() {
	if r := recover(); r != nil {
		if conf.LenStackBuf > 0 {
			buf := make([]byte, conf.LenStackBuf)
			l := runtime.Stack(buf, false)
			logging.Error(fmt.Sprintf("%v: %s", r, buf[:l]))
			log.Printf("%v: %s\n", r, buf[:l])
		} else {
			logging.Error(fmt.Sprintf("%v", r))
			log.Printf("%v\n", r)
		}
	}
}
