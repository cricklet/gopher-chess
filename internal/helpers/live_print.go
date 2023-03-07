package helpers

import (
	"fmt"
	"strings"
)

type LiveLogger struct {
	footer string
}

var _ Logger = &LiveLogger{}

func NewLiveLogger() *LiveLogger {
	fmt.Print("\033[B")
	fmt.Print("\033[B")
	l := &LiveLogger{footer: ""}
	PrintLive(Empty[string](), "", l.footer)
	return l
}

func (l *LiveLogger) Println(v ...interface{}) {
	l.Print(fmt.Sprintln(v...))
}

func (l *LiveLogger) Printf(format string, v ...interface{}) {
	l.Print(fmt.Sprintf(format, v...))
}

func (l *LiveLogger) Print(xs ...interface{}) {
	PrintLive(Some(fmt.Sprint(xs...)), l.footer, l.footer)
}

func (l *LiveLogger) SetFooter(s string) {
	PrintLive(Empty[string](), l.footer, s)
	l.footer = s
}

func PrintLive(output Optional[string], previousFooter string, footer string) {
	// > ... previous logging
	// > ^ caret is here for fmt.Println
	// (when println runs... we want to:
	//   print at the caret
	//   clear everything after
	// 	 and then reprint the live display at the bottom)

	for i := 0; i < len(strings.Split(previousFooter, "\n"))+1; i++ {
		fmt.Print("\033[A")
	}

	fmt.Print("\033[J")

	if output.HasValue() {
		fmt.Print(output.Value())
	}

	fmt.Println(footer)
	fmt.Println()

}
