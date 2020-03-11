package logging

import (
	"fmt"
	"log"
	"os"
	"time"
)

//參數設定
var (
	LogSavePath = "logs/"
	LogSaveName = "log"
	LogFileExt  = "log"
	TimeFormat  = "20060102"
	CurrentDate = time.Now().Format(TimeFormat)
)

func getLogFilePath() string {

	if len(os.Args) > 1 {
		LogSavePath = "logs/" + os.Args[1] + "/"
	}

	if len(os.Args) > 2 {
		LogSavePath = LogSavePath + os.Args[2] + "/"
	}

	return fmt.Sprintf("%s", LogSavePath)
}

func getLogFileFullPath() string {

	prefixPath := getLogFilePath()
	suffixPath := fmt.Sprintf("%s%s.%s", LogSaveName, CurrentDate, LogFileExt)

	return fmt.Sprintf("%s%s", prefixPath, suffixPath)
}

func openLogFile(filePath string) *os.File {
	_, err := os.Stat(filePath)
	switch {
	case os.IsNotExist(err):
		mkDir()
	case os.IsPermission(err):
		log.Fatalf("Permission :%v", err)
	}

	handle, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Fail to OpenFile :%v", err)
	}

	return handle
}

func mkDir() {
	dir, _ := os.Getwd()
	err := os.MkdirAll(dir+"/"+getLogFilePath(), os.ModePerm)
	if err != nil {
		panic(err)
	}
}

// checkFileExist 檢查檔案是否存在，如果不存在就重新開檔(會檢查兩次，第一次檢查檔案是否存在，第二次走原本流程)
func checkFileExist(filePath string) {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		openLogFile(filePath)
	}
}
