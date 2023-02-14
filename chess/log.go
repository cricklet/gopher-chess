package chess

import "log"

type Logger interface {
	Println(v ...any)
	Printf(format string, v ...any)
	Print(v ...any)
}

type DefaultLogger struct {
}

func (l *DefaultLogger) Println(v ...any) {
	log.Println(v...)
}
func (l *DefaultLogger) Printf(format string, v ...any) {
	log.Printf(format, v...)
}
func (l *DefaultLogger) Print(v ...any) {
	log.Print(v...)
}

var DEFAULT_LOGGER = DefaultLogger{}
