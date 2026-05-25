package config

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *zap.Logger

func InitLogger(environment string) {
	var core zapcore.Core

	if environment == "prod" {
		fileWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename:   "logs/prod.log",
			MaxSize:    10,
			MaxBackups: 5,
			MaxAge:     28,
			LocalTime:  true,
			Compress:   true,
		})

		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.TimeKey = "timestamp"
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			fileWriter,
			zap.InfoLevel,
		)
	} else {
		encoderConfig := zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

		fileWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename:   "logs/debug.log",
			MaxSize:    10,
			MaxBackups: 5,
			MaxAge:     28,
			LocalTime:  true,
			Compress:   true,
		})

		core = zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.NewMultiWriteSyncer(fileWriter, zapcore.AddSync(os.Stdout)),
			zapcore.DebugLevel,
		)
	}
	Log = zap.New(core, zap.AddCaller())
}
