package log

import (
	"encoding/json"
	"fmt"
	logging "github.com/shihray/gserver/logging/beego"
	"github.com/shihray/gserver/utils/conf"
)

func NewBeegoLogger(debug bool, ProcessID string, Logdir string, settings map[string]interface{}) *logging.BeeLogger {
	log := logging.NewLogger()
	log.ProcessID = ProcessID
	log.EnableFuncCallDepth(true)
	log.Async(1024) //同步打印,可能影响性能
	log.SetLogFuncCallDepth(4)
	if debug {
		//控制台
		log.SetLogger(logging.AdapterConsole)
	}
	if contenttype, ok := settings["contenttype"]; ok {
		log.SetContentType(contenttype.(string))
	}
	if f, ok := settings["file"]; ok {
		ff := f.(map[string]interface{})
		Prefix := ""
		if prefix, ok := ff["prefix"]; ok {
			Prefix = prefix.(string)
		}
		Suffix := ".log"
		if suffix, ok := ff["suffix"]; ok {
			Suffix = suffix.(string)
		}
		ff["filename"] = fmt.Sprintf("%s/%v%s%s", Logdir, Prefix, ProcessID, Suffix)
		config, err := json.Marshal(ff)
		if err != nil {
			logging.Error(err)
		}
		log.SetLogger(logging.AdapterFile, string(config))
	}
	if f, ok := settings["multifile"]; ok {
		multifile := f.(map[string]interface{})
		Prefix := ""
		if prefix, ok := multifile["prefix"]; ok {
			Prefix = prefix.(string)
		}
		Suffix := ".log"
		if suffix, ok := multifile["suffix"]; ok {
			Suffix = suffix.(string)
		}
		multifile["filename"] = fmt.Sprintf("%s/%v%s%s", Logdir, Prefix, ProcessID, Suffix)
		config, err := json.Marshal(multifile)
		if err != nil {
			logging.Error(err)
		}
		log.SetLogger(logging.AdapterMultiFile, string(config))
	}
	if dingtalk, ok := settings["dingtalk"]; ok {
		config, err := json.Marshal(dingtalk)
		if err != nil {
			logging.Error(err)
		}
		log.SetLogger(logging.AdapterDingtalk, string(config))
	}
	if slack, ok := settings["slack"]; ok {
		if conf.GetEnv("isLocal", "true") != "true" {
			_slack := slack.(map[string]interface{})
			config, err := json.Marshal(_slack)
			if err != nil {
				logging.Error(err)
			}
			log.SetLogger(logging.AdapterSlack, string(config))
		}
	}
	if jianliao, ok := settings["jianliao"]; ok {
		config, err := json.Marshal(jianliao)
		if err != nil {
			logging.Error(err)
		}
		log.SetLogger(logging.AdapterJianLiao, string(config))
	}
	if conn, ok := settings["conn"]; ok {
		config, err := json.Marshal(conn)
		if err != nil {
			logging.Error(err)
		}
		log.SetLogger(logging.AdapterConn, string(config))
	}
	if smtp, ok := settings["smtp"]; ok {
		config, err := json.Marshal(smtp)
		if err != nil {
			logging.Error(err)
		}
		log.SetLogger(logging.AdapterMail, string(config))
	}
	if es, ok := settings["es"]; ok {
		config, err := json.Marshal(es)
		if err != nil {
			logging.Error(err)
		}
		log.SetLogger(logging.AdapterEs, string(config))
	}
	return log
}
