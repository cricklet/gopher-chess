package helpers

import "log"

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
