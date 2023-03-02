package helpers

import (
	"fmt"
	"log"
)

type Logger interface {
	Println(v ...any)
	Printf(format string, v ...any)
	Print(v ...any)
}

type _defaultLogger struct {
}

func (l *_defaultLogger) Println(v ...any) {
	log.Println(v...)
}
func (l *_defaultLogger) Printf(format string, v ...any) {
	log.Printf(format, v...)
}
func (l *_defaultLogger) Print(v ...any) {
	log.Print(v...)
}

var DefaultLogger = _defaultLogger{}

type _silentLogger struct {
}

func (l *_silentLogger) Println(v ...any) {
}
func (l *_silentLogger) Printf(format string, v ...any) {
}
func (l *_silentLogger) Print(v ...any) {
}

var SilentLogger = _silentLogger{}

type _funcLogger struct {
	Callback func(string)
}

func FuncLogger(c func(string)) Logger {
	return &_funcLogger{c}
}

func (l *_funcLogger) Println(v ...any) {
	l.Callback(fmt.Sprintln(v...))
}
func (l *_funcLogger) Printf(format string, v ...any) {
	l.Callback(fmt.Sprintf(format, v...))
}
func (l *_funcLogger) Print(v ...any) {
	l.Callback(fmt.Sprint(v...))
}
