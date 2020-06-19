package conf

import (
	jsonIter "github.com/json-iterator/go"
	"io/ioutil"
)

var (
	LenStackBuf = 1024
	Conf        = Config{}
	JSONTool    = jsonIter.ConfigCompatibleWithStandardLibrary
)

type Config struct {
	Log      map[string]interface{}
	Rpc      Rpc
	Module   map[string][]*ModuleSettings
	Settings map[string]interface{}
}

type Rpc struct {
	MaxCoroutine int // 模塊同時可以創建的最大協程數量默認是100
	RpcExpired   int // 遠程訪問最後期限值 單位秒[默認5秒] 這個值指定了在客戶端可以等待服務端多長時間來應答
}

type ModuleSettings struct {
	ID       string
	Host     string
	Settings map[string]interface{}
}

// Read config.
func LoadConfig(Path string) {
	if err := readFileInto(Path); err != nil {
		panic(err)
	}
	if Conf.Rpc.MaxCoroutine == 0 {
		Conf.Rpc.MaxCoroutine = 100
	}
	if Conf.Rpc.RpcExpired == 0 {
		Conf.Rpc.RpcExpired = 1
	}
}

func readFileInto(path string) error {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return JSONTool.Unmarshal(bs, &Conf)
}
