package utils

import (
	"fmt"
	"runtime"

	logging "github.com/shihray/gserver/modules/logging"
	"github.com/shihray/gserver/modules/utils/conf"
)

func RecoverFunc() {
	if r := recover(); r != nil {
		if conf.LenStackBuf > 0 {
			buf := make([]byte, conf.LenStackBuf)
			l := runtime.Stack(buf, false)
			logging.Error(fmt.Sprintf("%v: %s", r, buf[:l]))
		} else {
			logging.Error(fmt.Sprintf("%v", r))
		}
	}
}
