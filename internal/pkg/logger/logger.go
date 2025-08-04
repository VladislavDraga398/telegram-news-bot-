package logger

import (
	"log"
	"os"
)

// Logger представляет собой логгер приложения
type Logger struct {
	infoLog  *log.Logger
	errorLog *log.Logger
}

// New создает новый экземпляр логгера
func New(prefix string) *Logger {
	return &Logger{
		infoLog:  log.New(os.Stdout, "["+prefix+" INFO] ", log.LstdFlags|log.Lmsgprefix),
		errorLog: log.New(os.Stderr, "["+prefix+" ERROR] ", log.LstdFlags|log.Lmsgprefix),
	}
}

// Info логирует информационное сообщение
func (l *Logger) Info(format string, v ...interface{}) {
	l.infoLog.Printf(format, v...)
}

// Error логирует сообщение об ошибке
func (l *Logger) Error(format string, v ...interface{}) {
	l.errorLog.Printf(format, v...)
}

// Fatal логирует критическую ошибку и завершает программу
func (l *Logger) Fatal(format string, v ...interface{}) {
	l.errorLog.Fatalf(format, v...)
}
