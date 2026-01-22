// Package logger is the standardized logger for all the ad-tech Go services
package logger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type ContextFields func(ctx context.Context) []zap.Field

/*
Logger & L are setup for convenient usage. L is created to reduce the stutter of using logger.Logger
instead use logger.L.Info(), logger.L.Error() etc.
*/
var (
	logHandler          *zap.Logger
	contextFieldsSetter ContextFields
)

// initializes zap logger
func init() { //nolint:gochecknoinits // it is essential for this package
	logHandler, _ = zap.NewProduction(zap.AddCallerSkip(1))
	zap.ReplaceGlobals(logHandler)
}

// ErrWithStacktrace logs an error with its stacktrace if available
func ErrWithStacktrace(err error) {
	logHandler.Error(fmt.Sprintf("%+v", err))
}

// Info logs a message with severity "info". This can be used for startup messages, exit messages etc. (on clean exit)
func Info(msg string, fields ...zap.Field) {
	logHandler.Info(msg, fields...)
}

// Debug logs a message with severity "debug".
func Debug(msg string, fields ...zap.Field) {
	logHandler.Debug(msg, fields...)
}

// Warn logs a message with severity "warn".
func Warn(msg string, fields ...zap.Field) {
	logHandler.Warn(msg, fields...)
}

// Error logs a message with severity "error"
func Error(msg string, fields ...zap.Field) {
	logHandler.Error(msg, fields...)
}

// Fatal logs a message with severity "fatal". This forces the app to exit with os.Exit(1)
func Fatal(msg string, fields ...zap.Field) {
	logHandler.Fatal(msg, fields...)
}

// SetGlobal overwrites the Global logHandler used in this package
func SetGlobal(zl *zap.Logger) {
	logHandler = zl
}

func SetContextFieldsSetter(fn ContextFields) {
	contextFieldsSetter = fn
}

func InfoCtx(ctx context.Context, msg string, fields ...zap.Field) {
	if contextFieldsSetter == nil {
		logHandler.Info(msg, fields...)
		return
	}

	logHandler.Info(msg, append(fields, contextFieldsSetter(ctx)...)...)
}

func DebugCtx(ctx context.Context, msg string, fields ...zap.Field) {
	if contextFieldsSetter == nil {
		logHandler.Debug(msg, fields...)
		return
	}

	logHandler.Debug(msg, append(fields, contextFieldsSetter(ctx)...)...)
}

func WarnCtx(ctx context.Context, msg string, fields ...zap.Field) {
	if contextFieldsSetter == nil {
		logHandler.Warn(msg, fields...)
		return
	}

	logHandler.Warn(msg, append(fields, contextFieldsSetter(ctx)...)...)
}

func ErrorCtx(ctx context.Context, msg string, fields ...zap.Field) {
	if contextFieldsSetter == nil {
		logHandler.Error(msg, fields...)
		return
	}

	logHandler.Error(msg, append(fields, contextFieldsSetter(ctx)...)...)
}

func FatalCtx(ctx context.Context, msg string, fields ...zap.Field) {
	if contextFieldsSetter == nil {
		logHandler.Fatal(msg, fields...)
		return
	}

	logHandler.Fatal(msg, append(fields, contextFieldsSetter(ctx)...)...)
}
