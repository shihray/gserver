package conf

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
)

var (
	LenStackBuf = 1024
	Conf        = Config{}
)

type Config struct {
	Log      map[string]interface{}
	Rpc      Rpc
	Module   map[string][]*ModuleSettings
	Settings map[string]interface{}
}

type Rpc struct {
	UDPMaxPacketSize int  // udp rpc 每一個包最大數據量 默認 4096
	MaxCoroutine     int  // 模塊同時可以創建的最大協程數量默認是100
	RpcExpired       int  // 遠程訪問最後期限值 單位秒[默認5秒] 這個值指定了在客戶端可以等待服務端多長時間來應答
	Log              bool // 是否打印RPC的日志
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
	var data []byte
	buf := new(bytes.Buffer)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadSlice('\n')
		if err != nil {
			if len(line) > 0 {
				buf.Write(line)
			}
			break
		}
		if !strings.HasPrefix(strings.TrimLeft(string(line), "\t "), "//") {
			buf.Write(line)
		}
	}
	data = buf.Bytes()
	//fmt.Print(string(data))
	return json.Unmarshal(data, &Conf)
}

// If read the file has an error,it will throws a panic.
func fileToStruct(path string, ptr *[]byte) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	*ptr = data
}
