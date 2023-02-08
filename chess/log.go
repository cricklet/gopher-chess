package chess

type Logger interface {
	Println(v ...any)
	Printf(format string, v ...any)
	Print(v ...any)
}
