package logger

import (
	log "github.com/sirupsen/logrus"
	"github.com/werbenhu/amqtt/config"
)

func init() {
	log.SetFormatter(&log.TextFormatter{})
	if config.Debug() {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func Debug(args ...interface{}) {
	log.Debug(args...)
}

func Trace(args ...interface{}) {
	log.Trace(args...)
}

func Info(args ...interface{}) {
	log.Info(args...)
}

func Warn(args ...interface{}) {
	log.Warn(args...)
}

func Error(args ...interface{}) {
	log.Error(args...)
}

func Fatal(args ...interface{}) {
	log.Fatal(args...)
}

func Panic(args ...interface{}) {
	log.Panic(args...)
}

func Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

func Tracef(format string, args ...interface{}) {
	log.Tracef(format, args...)
}

func Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	log.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

func Panicf(format string, args ...interface{}) {
	log.Panicf(format, args...)
}
