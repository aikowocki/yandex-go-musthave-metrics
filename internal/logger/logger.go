package logger

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(name string) (func(), error) {

	folder := fmt.Sprintf("logs/%s", name)
	filename := time.Now().Format("app-2006-01-02_15-04-05.log")

	err := os.MkdirAll(folder, 0755)
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(folder+"/"+filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return nil, err
	}

	encCfg := zap.NewDevelopmentEncoderConfig()

	var parsedLevel zapcore.Level
	logLevel := os.Getenv("LOG_LEVEL")

	if err := parsedLevel.UnmarshalText([]byte(logLevel)); err != nil {
		parsedLevel = zapcore.InfoLevel
	}

	consoleCore := zapcore.NewCore(zapcore.NewConsoleEncoder(encCfg), zapcore.AddSync(os.Stdout), zap.NewAtomicLevelAt(parsedLevel))
	fileCore := zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), zapcore.AddSync(file), zap.NewAtomicLevelAt(parsedLevel))

	teeCore := zapcore.NewTee(consoleCore, fileCore)

	zap.ReplaceGlobals(zap.New(teeCore))

	cleanup := func() {
		if err := zap.S().Sync(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "logger sync error: %v\n", err)
		}

		if err := file.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "log file close error: %v\n", err)
		}
	}

	return cleanup, nil
}
