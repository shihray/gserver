package basemodule

import (
	"sync"

	"github.com/shihray/gserver/modules/utils/conf"
)

type DefaultModule struct {
	mi       Module
	settings *conf.ModuleSettings
	closeSig chan bool
	wg       sync.WaitGroup
}

func run(m *DefaultModule) {
	defer utils.RecoverFunc()
	m.mi.Run(m.closeSig)
	m.wg.Done()
}

func destroy(m *DefaultModule) {
	defer utils.RecoverFunc()
	m.mi.OnDestroy()
}
