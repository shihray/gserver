package ping

import (
	module "github.com/shihray/gserver/module"
	basemodule "github.com/shihray/gserver/module/base"
	mqrpc "github.com/shihray/gserver/rpc"
	Conf "github.com/shihray/gserver/utils/conf"
	log "github.com/z9905080/gloger"
	"time"

	"github.com/shihray/gserver/utils/enum/module_type"
)

var Module = func() module.Module {
	this := new(Ping)
	return this
}

type Ping struct {
	basemodule.BaseModule
	updateStop bool // 結束更新
}

// version
func (p *Ping) GetType() string {
	return module_type.Ping.String()
}

// version
func (p *Ping) Version() string {
	return "1.0.0"
}

// initialize modules
func (p *Ping) OnInit(app module.App, settings *Conf.ModuleSettings) {
	p.BaseModule.OnInit(p, app, settings)
	p.updateStop = false

	p.GetServer().RegisterGO("PING", func(m map[string]interface{}) (string, string) {
		return "I'm PING, Return PONG", ""
	})

	p.GetServer().RegisterGO("HELLO", func(m map[string]interface{}) (string, string) {
		msg := p.GetModuleSettings().Settings["echo"].(string)

		if name, isExist := m["name"]; isExist {
			return "Hello" + name.(string) + " " + msg, ""
		}
		return "Hello you", ""
	})
}

func (p *Ping) Run(closeSig chan bool) {
	st := mqrpc.NewResultInvoke("PING", map[int]int{
		1: 1,
	}, 1, "123")

	go func() {
		tickUpdate := time.NewTicker(time.Duration(1) * time.Second)
		defer func() {
			tickUpdate.Stop()
		}()
		for !p.updateStop {
			select {
			case <-tickUpdate.C:
				{
					s, err := p.GetRandomServerByType("PONG")
					if err != nil {
						log.ErrorF("[%v]GetRandomServiceID Error :%v", s, err.Error())
					}
					//log.Error(s.CallNR(st))

					sID, getErr := p.GetRandomServiceID("PONG")
					if getErr != nil {
						log.ErrorF("[%v]GetRandomServiceID Error :%v", s, getErr.Error())
					}
					//log.Error()
					resp, getErr2 := p.RpcInvoke(sID, st)
					var tt struct {
						A int
					}
					Conf.JSONTool.Unmarshal(resp.([]byte), &tt)
					log.Error(resp, getErr2, tt)
				}
			}
		}
	}()
}

func (p *Ping) OnDestroy() {
	p.updateStop = true
	p.GetServer().OnDestroy()
}
