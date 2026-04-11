package core

import (
	"fmt"
	"os"
	"path"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var logsPath = "./logs"

const (
	LogColorReset = "\033[0m" // 重置颜色
	// 基础日志级别颜色
	LogColorDebug = "\033[37m"      // 灰色
	LogColorInfo  = "\033[36m"      // 青色
	LogColorWarn  = "\033[33m"      // 黄色
	LogColorError = "\033[31m"      // 红色
	LogColorFatal = "\033[35m"      // 洋红色
	LogColorPanic = "\033[31;1;43m" // 红字黄底粗体

	// 扩展级别颜色
	LogColorTrace    = "\033[90m" // 暗灰色
	LogColorNotice   = "\033[96m" // 亮青色
	LogColorCritical = "\033[91m" // 亮红色
	LogColorSuccess  = "\033[32m" // 绿色
)

type logEncoder struct {
	zapcore.Encoder
	errLog      *os.File
	logFile     *os.File
	currentTime string
	Service     string
}

var Encoder logEncoder

func LevelEncoder(l zapcore.Level, p zapcore.PrimitiveArrayEncoder) {
	var color string
	switch l {
	case zapcore.DebugLevel:
		color = LogColorDebug
	case zapcore.InfoLevel:
		color = LogColorInfo
	case zapcore.WarnLevel:
		color = LogColorWarn
	case zapcore.ErrorLevel:
		color = LogColorError
	case zapcore.DPanicLevel:
		color = LogColorPanic
	case zapcore.PanicLevel:
		color = LogColorPanic
	case zapcore.FatalLevel:
		color = LogColorFatal
	default:
		if l < zapcore.DebugLevel {
			color = LogColorTrace
		} else if l > zapcore.FatalLevel {
			color = LogColorCritical
		} else {
			color = LogColorInfo
		}
		p.AppendString(l.String())
		return
	}
	p.AppendString(fmt.Sprintf("%s%s%s", color, l.String(), LogColorReset))
}

func (l *logEncoder) EncodeEntry(enc zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buff, err := l.Encoder.EncodeEntry(enc, fields)
	if err != nil {
		panic(err)
	}

	data := buff.String()
	buff.Reset()
	buff.WriteString(fmt.Sprintf("%s[%s]%s %s", LogColorSuccess, l.Service, LogColorReset, data))

	now := time.Now().Format("2006-01-02/15")
	if l.currentTime != now {
		l.currentTime = now
		logPath := path.Join(logsPath, now)
		os.MkdirAll(logPath, os.ModePerm)
		if file, err := os.Create(path.Join(logPath, "info.log")); err != nil {
			panic(err)
		} else {
			l.logFile = file
		}

		if file, err := os.Create(path.Join(logPath, "error.log")); err != nil {
			panic(err)
		} else {
			l.errLog = file
		}
	}
	l.logFile.WriteString(data)
	if enc.Level > zapcore.WarnLevel {
		l.errLog.WriteString(data)
	}
	return buff, nil
}

func InitLogger() {
	Encoder.Service = "fast-gin"
	os.MkdirAll(logsPath, os.ModePerm)
	developmentConfig := zap.NewDevelopmentConfig()
	developmentConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	developmentConfig.EncoderConfig.EncodeLevel = LevelEncoder
	Encoder.Encoder = zapcore.NewConsoleEncoder(developmentConfig.EncoderConfig)
	logger := CustomLogger(&Encoder, os.Stdout, zap.DebugLevel, zap.AddCaller())
	zap.ReplaceGlobals(logger)
	zap.L().Info("zap logger 初始化成功")
}

func CustomLogger(enc *logEncoder, ws zapcore.WriteSyncer, enab zapcore.LevelEnabler, options ...zap.Option) *zap.Logger {
	core := zapcore.NewCore(enc, ws, enab)
	logger := zap.New(core, options...)
	return logger
}
