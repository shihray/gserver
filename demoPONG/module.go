package pong

import (
	module "github.com/shihray/gserver/module"
	basemodule "github.com/shihray/gserver/module/base"
	mqrpc "github.com/shihray/gserver/rpc"
	Conf "github.com/shihray/gserver/utils/conf"
	"github.com/shihray/gserver/utils/enum/moduleType"
	log "github.com/z9905080/gloger"
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

	p.GetServer().RegisterGO("PING", func(m map[string]interface{}) (string, string) {

		st := mqrpc.NewResultInvoke("HELLO", map[string]interface{}{
			"name": "123",
		})
		if res, err := p.RpcInvoke("PING", st); err != "" {
			log.Debug(err)
		} else {
			log.DebugF("%v: %v", p.GetType(), res)
		}

		return "I'm PONG, Return PING", ""
	})
}

func (p *Pong) Run(closeSig chan bool) {
}

func (p *Pong) OnDestroy() {
	p.updateStop = true
	p.GetServer().OnDestroy()
}
