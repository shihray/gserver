package logging

import (
	"log"
	"os"
	"time"
)

// DeleteOldLog 刪除舊的log
func DeleteOldLog() error {

	var deleltFolderList = []string{
		"game_module/brnn",
		"game_module/dt",
		"game_module/cl",
		"game_module/bacc",
		"game_module/dgp",
		"master_server",
		"game_server",
		"rpc_server",
	}

	logPath := ""
	daylist := GetDayBeforeMonth()
	for i := 0; i < 30; i++ {
		for _, folder := range deleltFolderList {
			logPath = "logs/" + folder + "/log" + daylist[i] + ".log"
			executeDeleteFile(logPath)
		}
	}
	return nil
}

// GetDayBeforeMonth 取一個月以前的日期列表
func GetDayBeforeMonth() []string {

	var dayList = make([]string, 30)
	for i := 0; i < 30; i++ {
		// 調整時間
		// +8 : 台北時間, -24*10 : 保留10天：依序往前數30天
		ajustedTime := time.Now().UTC().Add(time.Duration(8-24*10-24*i) * time.Hour)
		day := ajustedTime.Format("20060102")
		dayList[i] = day
	}
	return dayList
}

// executeDeleteFile 實際執行刪除檔案
func executeDeleteFile(logPath string) {
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		// 沒找到檔案就不動作
	} else {
		removeErr := os.RemoveAll(logPath)
		if removeErr != nil {
			log.Println(removeErr.Error())
		}
	}
}
