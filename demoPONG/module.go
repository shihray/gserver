package pong

import (
	"fmt"
	mqrpc "github.com/shihray/gserver/rpc"
	"github.com/shihray/gserver/utils/enum/moduleType"

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

// version
func (p *Pong) GetType() string {
	return moduleType.Pong.String()
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
	st := mqrpc.NewResultInvoke("HELLO", map[string]interface{}{
		"name": "123",
	})

	if res, err := p.RpcInvoke("PING", st); err != "" {
		fmt.Println(err)
	} else {
		fmt.Println(p.GetType(), ":", res)
	}
}

func (p *Pong) OnDestroy() {
	p.updateStop = true
	p.GetServer().OnDestroy()
}
