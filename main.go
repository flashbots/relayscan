package main

import (
	"flag"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	version = "dev" // is set during build process

	// Default values
	defaultDebug      = os.Getenv("DEBUG") == "1"
	defaultLogProd    = os.Getenv("LOG_PROD") == "1"
	defaultLogService = os.Getenv("LOG_SERVICE")

	// Flags
	debugPtr      = flag.Bool("debug", defaultDebug, "print debug output")
	logProdPtr    = flag.Bool("log-prod", defaultLogProd, "log in production mode (json)")
	logServicePtr = flag.String("log-service", defaultLogService, "'service' tag to logs")
)

func main() {
	flag.Parse()

	logger, _ := zap.NewDevelopment()
	if *logProdPtr {
		atom := zap.NewAtomicLevel()
		if *debugPtr {
			atom.SetLevel(zap.DebugLevel)
		}

		encoderCfg := zap.NewProductionEncoderConfig()
		encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		logger = zap.New(zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.Lock(os.Stdout),
			atom,
		))
	}
	defer func() { _ = logger.Sync() }()
	log := logger.Sugar()

	if *logServicePtr != "" {
		log = log.With("service", *logServicePtr)
	}

	log.Info("Starting your-project", "version", version)

	log.Debug("debug message")
	log.Info("info message")
	log.Warn("warn message")
	log.Error("error message with, trace in dev mode")
	// log.Fatal("fatal message")
}
