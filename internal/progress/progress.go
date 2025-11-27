package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type Writer struct {
	out         io.Writer
	mu          sync.Mutex
	startTime   time.Time
	lastStatus  string
	dotCount    int
	stopChan    chan struct{}
	doneChan    chan struct{}
	currentLine string
	animating   bool
}

func New() *Writer {
	return &Writer{
		out:       os.Stdout,
		startTime: time.Now(),
		stopChan:  make(chan struct{}),
		doneChan:  make(chan struct{}),
	}
}

func (w *Writer) SetStartTime(t time.Time) {
	w.startTime = t
}

func (w *Writer) Elapsed() time.Duration {
	return time.Since(w.startTime)
}

func (w *Writer) FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func (w *Writer) clearLine() {
	fmt.Fprint(w.out, "\r\033[K")
}

func (w *Writer) Header(format string, args ...interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	fmt.Fprintf(w.out, format+"\n", args...)
}

func (w *Writer) Status(status string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if status != w.lastStatus {
		if w.animating {
			w.clearLine()
		}
		fmt.Fprintf(w.out, "\nStatus: %s\n", status)
		w.lastStatus = status
	}
}

func (w *Writer) Action(format string, args ...interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.animating {
		w.clearLine()
	}
	fmt.Fprintf(w.out, "→ "+format+"\n", args...)
}

func (w *Writer) Success(format string, args ...interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.animating {
		w.clearLine()
	}
	fmt.Fprintf(w.out, "✓ "+format+"\n", args...)
}

func (w *Writer) Error(format string, args ...interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.animating {
		w.clearLine()
	}
	fmt.Fprintf(w.out, "✗ "+format+"\n", args...)
}

func (w *Writer) StartWait(message string, statsFunc func() string) {
	w.mu.Lock()
	w.animating = true
	w.dotCount = 0
	w.stopChan = make(chan struct{})
	w.doneChan = make(chan struct{})
	w.mu.Unlock()

	go func() {
		defer close(w.doneChan)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-w.stopChan:
				return
			case <-ticker.C:
				w.mu.Lock()
				w.dotCount = (w.dotCount % 3) + 1
				dots := strings.Repeat(".", w.dotCount)
				elapsed := w.FormatDuration(w.Elapsed())

				stats := ""
				if statsFunc != nil {
					stats = statsFunc()
				}

				line := fmt.Sprintf("%s %s (%s)", dots, message, elapsed)
				if stats != "" {
					line += " | " + stats
				}

				w.clearLine()
				fmt.Fprint(w.out, line)
				w.currentLine = line
				w.mu.Unlock()
			}
		}
	}()
}

func (w *Writer) StopWait() {
	w.mu.Lock()
	if !w.animating {
		w.mu.Unlock()
		return
	}
	w.animating = false
	close(w.stopChan)
	w.mu.Unlock()

	<-w.doneChan

	w.mu.Lock()
	w.clearLine()
	w.mu.Unlock()
}

func (w *Writer) TotalTime() string {
	return w.FormatDuration(w.Elapsed())
}
