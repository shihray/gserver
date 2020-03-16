package ping

import (
	"fmt"
	"time"

	module "github.com/shihray/gserver/module"
	basemodule "github.com/shihray/gserver/module/base"
	Conf "github.com/shihray/gserver/utils/conf"
)

var Module = func() module.Module {
	this := new(Ping)
	return this
}

type Ping struct {
	basemodule.BaseModule
	updateStop bool // 結束更新
}

// same as config file modules name
func (p *Ping) GetType() string {
	return "PING"
}

// version
func (p *Ping) Version() string {
	return "1.0.0"
}

// initialize modules
func (p *Ping) OnInit(app module.App, settings *Conf.ModuleSettings) {
	p.BaseModule.OnInit(p, app, settings)
	p.updateStop = false

	p.GetServer().Register("PING", func(m map[string]interface{}) (string, string) {
		return "I'm PING, Return PONG", ""
	})

	fmt.Println("Ping OnInit...")
}

func (p *Ping) Run(closeSig chan bool) {
	go func() {
		tickUpdate := time.NewTicker(time.Duration(1) * time.Second)
		defer func() {
			tickUpdate.Stop()
		}()
		for !p.updateStop {
			select {
			case <-tickUpdate.C:
				if res, err := p.RpcInvoke("PONG", "PING", nil); err != "" {
					fmt.Println(err)
				} else {
					fmt.Println(res)
				}
			}
		}
	}()
}

func (p *Ping) OnDestroy() {
	p.updateStop = true
	p.GetServer().OnDestroy()
}
