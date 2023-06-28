package helpers

import (
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/acarl005/stripansi"
)

type LiveLogger struct {
	footers []string

	lock   sync.Mutex
	noCopy NoCopy
}

var _ Logger = &LiveLogger{}

func NewLiveLogger() *LiveLogger {
	fmt.Print("\033[B")
	l := &LiveLogger{footers: []string{}}
	l.PrintLive(Empty[string](), l.FooterString(), l.FooterString())
	return l
}

type _footerLogger struct {
	logger *LiveLogger
	i      int
}

func NewFooterLogger(logger *LiveLogger, i int) *_footerLogger {
	return &_footerLogger{logger: logger, i: i}
}

func (l *_footerLogger) Println(v ...any) {
	l.logger.SetFooter(fmt.Sprintln(v...), l.i)
}
func (l *_footerLogger) Printf(format string, v ...any) {
	l.logger.SetFooter(fmt.Sprintf(format, v...), l.i)
}
func (l *_footerLogger) Print(v ...any) {
	l.logger.SetFooter(fmt.Sprint(v...), l.i)
}

var FooterLogger = _footerLogger{}

func (l *LiveLogger) FooterString() string {
	return strings.Join(l.footers, "\n")
}

func (l *LiveLogger) FlushFooter() {
	l.Println(l.FooterString())
}

func (l *LiveLogger) Println(v ...interface{}) {
	l.Print(fmt.Sprintln(v...))
}

func (l *LiveLogger) Printf(format string, v ...interface{}) {
	l.Print(fmt.Sprintf(format, v...))
}

func (l *LiveLogger) Print(xs ...interface{}) {
	l.PrintLive(Some(fmt.Sprint(xs...)), l.FooterString(), l.FooterString())
}

func runeCountIgnoringAnsi(s string) int {
	return utf8.RuneCountInString(stripansi.Strip(s))
}

func wrapLine(s string, width int) string {
	if runeCountIgnoringAnsi(s) < width {
		return s
	}

	words := strings.Split(s, " ")
	lines := []string{}
	line := []string{}
	for _, word := range words {
		joinedLine := strings.Join(line, " ")
		if runeCountIgnoringAnsi(joinedLine)+runeCountIgnoringAnsi(word)+1 > width && len(line) != 0 {
			lines = append(lines, joinedLine)
			line = []string{word}
		} else {
			line = append(line, word)
		}
	}
	lines = append(lines, strings.Join(line, " "))
	return strings.Join(
		MapSlice(lines, func(s string) string { return strings.TrimSpace(s) }), "\n")
}

func wrapText(s string, width int, indent string) string {
	result := []string{}
	for _, line := range strings.Split(s, "\n") {
		result = append(result, Indent(wrapLine(line, width), indent))
	}

	return strings.Join(result, "\n")
}

func (l *LiveLogger) SetFooter(s string, index int) {
	// s = wrapText(s, termWidth(), "  ")
	s = strings.TrimSpace(s)

	prevFooterString := l.FooterString()

	for i := len(l.footers) - 1; i < index; i++ {
		l.footers = append(l.footers, "")
	}
	l.footers[index] = s

	l.PrintLive(Empty[string](), prevFooterString, l.FooterString())
}

func (l *LiveLogger) PrintLive(output Optional[string], previousFooter string, footer string) {
	// > ... previous logging
	// > ^ caret is here for fmt.Println
	// (when println runs... we want to:
	//   print at the caret
	//   clear everything after
	// 	 and then reprint the live display at the bottom)

	l.lock.Lock()
	defer l.lock.Unlock()

	for i := 0; i < len(strings.Split(previousFooter, "\n")); i++ {
		fmt.Print("\033[A")
	}

	fmt.Print("\033[J")

	if output.HasValue() {
		fmt.Print(output.Value())
	}

	fmt.Println(footer)
}
