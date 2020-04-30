package log

import (
	beeGoLog "github.com/shihray/gserver/logging/beego"
)

var beego *beeGoLog.BeeLogger

func InitLog(debug bool, logDir string, settings map[string]interface{}) {
	beego = NewBeegoLogger(debug, logDir, settings)
}

func LogBeego() *beeGoLog.BeeLogger {
	if beego == nil {
		beego = beeGoLog.NewLogger()
	}
	return beego
}

func CreateTrace(trace, span string) TraceSpan {
	return &TraceSpanImp{
		Trace: trace,
		Span:  span,
	}
}

func Debug(format string, a ...interface{}) {
	//gLogger.doPrintf(debugLevel, printDebugLevel, format, a...)
	LogBeego().Debug(nil, format, a...)
}

func Info(format string, a ...interface{}) {
	//gLogger.doPrintf(releaseLevel, printReleaseLevel, format, a...)
	LogBeego().Info(nil, format, a...)
}

func Notice(format string, a ...interface{}) {
	LogBeego().Notice(nil, format, a...)
}

func Error(format string, a ...interface{}) {
	//gLogger.doPrintf(errorLevel, printErrorLevel, format, a...)
	LogBeego().Error(nil, format, a...)
}

func Warning(format string, a ...interface{}) {
	//gLogger.doPrintf(fatalLevel, printFatalLevel, format, a...)
	LogBeego().Warning(nil, format, a...)
}

func Close() {
	LogBeego().Close()
}
