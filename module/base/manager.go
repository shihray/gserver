package basemodule

import (
	"fmt"
	module "github.com/shihray/gserver/module"
	"github.com/shihray/gserver/utils/conf"
)

func NewModuleManager() (m *ModuleManager) {
	m = new(ModuleManager)
	return
}

type ModuleManager struct {
	app     module.App
	mods    []*DefaultModule
	runMods []*DefaultModule
}

func (mer *ModuleManager) Register(mi module.Module) {
	md := new(DefaultModule)
	md.mi = mi
	md.closeSig = make(chan bool, 1)

	mer.mods = append(mer.mods, md)
}

func (mer *ModuleManager) RegisterRunMod(mi module.Module) {
	md := new(DefaultModule)
	md.mi = mi
	md.closeSig = make(chan bool, 1)

	mer.runMods = append(mer.runMods, md)
}

func (mer *ModuleManager) Init(app module.App) {
	mer.app = app
	mer.CheckModuleSettings() // 配置文件規則檢查
	for i := 0; i < len(mer.mods); i++ {
		for Type, modSettings := range app.GetSettings().Module {
			if mer.mods[i].mi.GetType() == Type {
				mer.runMods = append(mer.runMods, mer.mods[i]) // 這裏加入能夠運行的組件
				for _, setting := range modSettings {
					mer.mods[i].settings = setting
				}
				break // 跳出內部循環
			}
		}
	}

	for i := 0; i < len(mer.runMods); i++ {
		m := mer.runMods[i]
		m.mi.OnInit(app, m.settings)

		if app.GetModuleInit() != nil {
			app.GetModuleInit()(app, m.mi)
		}

		m.wg.Add(1)
		go run(m)
	}
}

/**
module配置文件規則檢查
1. ID全局必須唯一
*/
func (mer *ModuleManager) CheckModuleSettings() {
	// 用來保存全局ID-ModuleType
	gID := map[string]string{}
	for Type, modSettings := range conf.Conf.Module {
		for _, setting := range modSettings {
			if Stype, ok := gID[setting.ID]; ok {
				// 如果ID已經存在,說明有兩個相同ID的模塊,這種情況不能被允許,這裏就直接拋異常 強制崩潰以免以後調試找不到問題
				panic(fmt.Sprintf("ID (%s) been used in modules of type [%s] and cannot be reused", setting.ID, Stype))
			} else {
				gID[setting.ID] = Type
			}
		}
	}
}

func (mer *ModuleManager) Destroy() {
	for i := len(mer.runMods) - 1; i >= 0; i-- {
		m := mer.runMods[i]
		m.closeSig <- true
		m.wg.Wait()
		destroy(m)
	}
}
