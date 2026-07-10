package downloader

import "go.uber.org/zap"

func newLogger() *zap.SugaredLogger {
	logger, _ := zap.NewProduction()
	return logger.Sugar()
}
