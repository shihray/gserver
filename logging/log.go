package log

import (
	beeGoLog "github.com/shihray/gserver/logging/beego"
)

var beego *beeGoLog.BeeLogger
var bi *beeGoLog.BeeLogger

func InitLog(debug bool, ProcessID string, logDir string, settings map[string]interface{}) {
	beego = NewBeegoLogger(debug, ProcessID, logDir, settings)
}

func InitBI(debug bool, ProcessID string, logDir string, settings map[string]interface{}) {
	bi = NewBeegoLogger(debug, ProcessID, logDir, settings)
}

func LogBeego() *beeGoLog.BeeLogger {
	if beego == nil {
		beego = beeGoLog.NewLogger()
	}
	return beego
}

func BiBeego() *beeGoLog.BeeLogger {
	return bi
}

func CreateTrace(trace, span string) TraceSpan {
	return &TraceSpanImp{
		Trace: trace,
		Span:  span,
	}
}

func BiReport(msg string) {
	//gLogger.doPrintf(debugLevel, printDebugLevel, format, a...)
	l := BiBeego()
	if l != nil {
		l.BiReport(msg)
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

func TDebug(span TraceSpan, format string, a ...interface{}) {
	if span != nil {
		LogBeego().Debug(
			&beeGoLog.BeegoTraceSpan{
				Trace: span.TraceId(),
				Span:  span.SpanId(),
			}, format, a...)
	} else {
		LogBeego().Debug(nil, format, a...)
	}
}

func TInfo(span TraceSpan, format string, a ...interface{}) {
	if span != nil {
		LogBeego().Info(
			&beeGoLog.BeegoTraceSpan{
				Trace: span.TraceId(),
				Span:  span.SpanId(),
			}, format, a...)
	} else {
		LogBeego().Info(nil, format, a...)
	}
}

func TError(span TraceSpan, format string, a ...interface{}) {
	if span != nil {
		LogBeego().Error(
			&beeGoLog.BeegoTraceSpan{
				Trace: span.TraceId(),
				Span:  span.SpanId(),
			}, format, a...)
	} else {
		LogBeego().Error(nil, format, a...)
	}
}

func TWarning(span TraceSpan, format string, a ...interface{}) {
	if span != nil {
		LogBeego().Warning(
			&beeGoLog.BeegoTraceSpan{
				Trace: span.TraceId(),
				Span:  span.SpanId(),
			}, format, a...)
	} else {
		LogBeego().Warning(nil, format, a...)
	}
}

func Close() {
	LogBeego().Close()
}
