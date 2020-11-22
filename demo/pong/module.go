package pong

import (
	module "github.com/shihray/gserver/module"
	basemodule "github.com/shihray/gserver/module/base"
	Conf "github.com/shihray/gserver/utils/conf"
	"github.com/shihray/gserver/utils/enum/module_type"
	"log"
	"strconv"
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
	return module_type.Pong.String()
}

// version
func (p *Pong) Version() string {
	return "1.0.0"
}

// initialize modules
func (p *Pong) OnInit(app module.App, settings *Conf.ModuleSettings) {
	p.BaseModule.OnInit(p, app, settings)
	p.updateStop = false

	p.GetServer().RegisterGO("TEST", func(a int) (string, string) {
		return strconv.Itoa(a), ""
	})

	p.GetServer().RegisterGO("PING", func(m interface{}, aa int, b string) (interface{}, string) {

		//st := mqrpc.NewResultInvoke("HELLO", map[string]interface{}{
		//	"name": "123",
		//})
		//if res, err := p.RpcInvoke("PING", st); err != "" {
		//	log.Debug(err)
		//} else {
		//	log.DebugF("%v: %v", p.GetType(), res)
		//}
		log.Println(m, aa, b)

		a := struct {
			A int
		}{
			A: 1,
		}
		return a, ""
	})
}

func (p *Pong) Run(closeSig chan bool) {
}

func (p *Pong) OnDestroy() {
	p.updateStop = true
	p.GetServer().OnDestroy()
}
