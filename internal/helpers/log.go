package helpers

import (
	"fmt"
)

type Logger interface {
	Println(v ...any)
	Printf(format string, v ...any)
	Print(v ...any)
}

type _defaultLogger struct {
}

var ellipseCutoff = 120

func (l *_defaultLogger) Println(v ...any) {
	fmt.Println(Ellipses(fmt.Sprint(v...), ellipseCutoff))
	// fmt.Println(v...)
}
func (l *_defaultLogger) Printf(format string, v ...any) {
	fmt.Print(Ellipses(fmt.Sprintf(format, v...), ellipseCutoff))
	// fmt.Printf(format, v...)
}
func (l *_defaultLogger) Print(v ...any) {
	fmt.Print(Ellipses(fmt.Sprint(v...), ellipseCutoff))
	// fmt.Print(v...)
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
