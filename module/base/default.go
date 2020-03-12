package basemodule

import (
	"sync"

	module "github.com/shihray/gserver/module"
	"github.com/shihray/gserver/utils"
	"github.com/shihray/gserver/utils/conf"
)

type DefaultModule struct {
	mi       module.Module
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
