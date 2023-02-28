package helpers

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/dustin/go-humanize"
	"golang.org/x/term"
)

type ProgressBar struct {
	Set   func(int)
	Add   func(int)
	Close func()
}

// func CreateProgressBar(total int, label string) ProgressBar {
// 	p := bar.New(total)
// 	return ProgressBar{
// 		func(i int) {
// 			p.Tick()
// 		}, func(i int) {
// 			p.Tick()
// 		}, func() {
// 			p.Done()
// 		},
// 	}
// }

// func CreateProgressBar(total int, label string) ProgressBar {
// 	p := uiprogress.AddBar(total)
// 	return ProgressBar{
// 		func(i int) {
// 			_ = p.Set(p.Total + i)
// 		}, func(i int) {
// 			_ = p.Set(i)
// 		}, func() {

// 		},
// 	}
// }

// func CreateProgressBar(total int, label string) ProgressBar {
// 	p := progressbar.Default(int64(total), label)
// 	return ProgressBar{
// 		func(i int) {
// 			_ = p.Add(i)
// 		}, func(i int) {
// 			_ = p.Set(i)
// 		}, func() {
// 			p.Close()
// 		},
// 	}
// }

func termWidth() int {
	width, _, err := term.GetSize(0)
	if !IsNil(err) {
		return 80
	}
	return MaxInt(80, MinInt(120, width))
}

func unitForDuration(d time.Duration) time.Duration {
	if d < time.Microsecond {
		return time.Nanosecond
	}
	if d < time.Millisecond {
		return time.Microsecond
	}
	if d < time.Second {
		return time.Millisecond
	}
	if d < time.Minute {
		return time.Second
	}
	return time.Minute
}

func CreateProgressBar(total int, label string) ProgressBar {
	value := int32(0)

	startTime := time.Now()
	updateDuration := time.Millisecond * 200

	var update = func(forceUpdate bool) {
		shouldUpdate := false
		if time.Since(startTime) > updateDuration || forceUpdate {
			shouldUpdate = true
			updateDuration *= 2
		}

		if shouldUpdate {
			elapsed := time.Since(startTime)

			perSecond := int(float64(value) / elapsed.Seconds())

			if value > int32(total) {
				value = int32(total)
			} else if value == 0 {
				return
			}

			percent := float64(value) / float64(total)
			percentStr := fmt.Sprintf("%3d", int(percent*100))
			expectedFinish := time.Duration(float64(elapsed) / percent)
			unit := unitForDuration(elapsed)

			prefix := fmt.Sprintf("%s %s%% ", label, percentStr)
			suffix := fmt.Sprintf(" %v => %v @ %v/s", elapsed.Round(unit), expectedFinish.Round(unit), humanize.Comma(int64(perSecond)))

			width := termWidth()
			textLen := utf8.RuneCountInString(prefix) + utf8.RuneCountInString(suffix)
			totalProgressLen := width - textLen
			currentProgressLen := MinInt(MaxInt(int(float64(totalProgressLen)*percent), 0), totalProgressLen)
			remainingProgressLen := totalProgressLen - currentProgressLen

			fmt.Printf("%s%s%s%s\n", prefix, strings.Repeat("=", currentProgressLen), strings.Repeat(" ", remainingProgressLen), suffix)
		}
	}
	return ProgressBar{
		func(i int) {
			atomic.StoreInt32(&value, int32(i))
			update(false)
		},
		func(i int) {
			atomic.AddInt32(&value, int32(i))
			update(false)
		}, func() {
			update(true)
		},
	}
}

// func CreateProgressBar(total int, label string) ProgressBar {
// 	p := pb.StartNew(total)
// 	return ProgressBar{
// 		func(i int) {
// 			_ = p.Add(i)
// 		}, func(i int) {
// 			_ = p.SetCurrent(int64(i))
// 		}, func() {
// 			p.Finish()
// 		},
// 	}
// }
