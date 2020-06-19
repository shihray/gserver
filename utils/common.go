package utils

import (
	"runtime"

	"github.com/shihray/gserver/utils/conf"
	log "github.com/z9905080/gloger"
)

const Nano2Millisecond int64 = 1000000

func RecoverFunc() {
	if r := recover(); r != nil {
		buf := make([]byte, conf.LenStackBuf)
		l := runtime.Stack(buf, false)
		log.ErrorF("%v: %s", r, buf[:l])
	}
}
