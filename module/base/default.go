package basemodule

import (
	"errors"
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

type ModuleErr struct {
	ID   int
	Name string
	Err  error
}

var (
	ServiceNotFound = ModuleErr{ID: 001, Name: "service not found", Err: errors.New("service not found")}
)

func (e ModuleErr) String() string {
	return e.Name
}

func (e ModuleErr) Int() int {
	return e.ID
}

func (e ModuleErr) Error() error {
	return e.Err
}
