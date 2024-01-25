package app

import (
	"os"

	"fmt"
	"strings"

	"github.com/hibrid/statemachine2_transactional_kafka/lib/env"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Name holds the application name
var Name = os.Args[0]

// Log holds a reference to our configured structured logger
var Log *zap.Logger

// SiteHTTPPort holds the env var value for which http port to listen on
var SiteHTTPPort = env.Get("HTTP_LISTEN")

func init() {
	cfg := zap.NewProductionConfig()
	cfg.Build()
	var logLevel zapcore.Level
	err := logLevel.UnmarshalText([]byte(strings.ToLower(env.Get("LOG_LEVEL"))))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot setup logger, reason:", err)
		os.Exit(1)
	}

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeDuration = zapcore.StringDurationEncoder
	encCfg.StacktraceKey = "stack"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewJSONEncoder(encCfg)

	out := zapcore.Lock(os.Stdout)

	core := zapcore.NewCore(encoder, out, zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= logLevel
	}))
	fields := []zapcore.Field{}
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	fields = append(fields, zap.String("app", Name))
	if env.IsProdLike {
		fields = append(fields, zap.String("host", hostname))
	}

	Log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)).With(fields...)

	zap.RedirectStdLog(Log.With(zap.Bool("std", true)))
}

// Logger returns the configure logger
func Logger() *zap.Logger {
	return Log
}
