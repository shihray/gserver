package pong

import (
	"fmt"

	module "github.com/shihray/gserver/module"
	basemodule "github.com/shihray/gserver/module/base"
	Conf "github.com/shihray/gserver/utils/conf"
)

var Module = func() module.Module {
	this := new(Pong)
	return this
}

type Pong struct {
	basemodule.BaseModule
	updateStop bool // 結束更新
}

// same as config file modules name
func (p *Pong) GetType() string {
	return "PONG"
}

// version
func (p *Pong) Version() string {
	return "1.0.0"
}

// initialize modules
func (p *Pong) OnInit(app module.App, settings *Conf.ModuleSettings) {
	p.BaseModule.OnInit(p, app, settings)
	p.updateStop = false

	p.GetServer().Register("PING", func(m map[string]interface{}) (string, string) {
		return "I'm PONG, Return PING", ""
	})

	fmt.Println("Pong OnInit...")
}

func (p *Pong) Run(closeSig chan bool) {
}

func (p *Pong) OnDestroy() {
	p.updateStop = true
	p.GetServer().OnDestroy()
}
