package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// A sync is seconds of network - measured at half a minute on a real queue - and it runs
// with the UI torn down, because the UI never acts: it records the key and quits, and the
// caller does the work. So for that whole time the float holds a blank screen unless
// something here draws.
//
// One line, rewritten in place and erased when the work lands: the project, every leg of
// the sync with a ✓ as it returns, and the seconds so far. Naming the legs is the point -
// "syncing…" for thirty seconds says nothing about whether it is stuck.
type progressLine struct {
	mu      sync.Mutex
	title   string
	order   []string
	done    map[string]bool
	count   map[string]int
	started time.Time
	stop    chan struct{}
	wg      sync.WaitGroup
	drawing bool
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// newProgressLine starts drawing. On anything that is not a terminal it stays silent, so
// `workdesk sync` in a script or a test writes only its result.
func newProgressLine(title string) *progressLine {
	p := &progressLine{
		title:   title,
		done:    map[string]bool{},
		count:   map[string]int{},
		started: time.Now(),
		stop:    make(chan struct{}),
		drawing: isTTY(),
	}
	if !p.drawing {
		return p
	}
	p.wg.Go(func() {
		tick := time.NewTicker(120 * time.Millisecond)
		defer tick.Stop()
		for frame := 0; ; frame++ {
			select {
			case <-p.stop:
				return
			case <-tick.C:
				p.draw(frame)
			}
		}
	})
	return p
}

// leg is the workdesk.Progress callback. The fetches run together, so it is called from
// several goroutines at once.
func (p *progressLine) leg(name string, done bool, n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, seen := p.done[name]; !seen {
		p.order = append(p.order, name)
	}
	p.done[name] = done
	p.count[name] = n
}

func (p *progressLine) draw(frame int) {
	p.mu.Lock()
	parts := make([]string, 0, len(p.order))
	for _, name := range p.order {
		switch {
		case p.done[name] && p.count[name] > 0:
			parts = append(parts, fmt.Sprintf("%s %d ✓", name, p.count[name]))
		case p.done[name]:
			parts = append(parts, name+" ✓")
		default:
			parts = append(parts, name+"…")
		}
	}
	p.mu.Unlock()

	line := fmt.Sprintf("%s %s", spinnerFrames[frame%len(spinnerFrames)], p.title)
	if len(parts) > 0 {
		line += " · " + strings.Join(parts, " · ")
	}
	// \r to the start and \033[K to the end of line: the line is replaced, never stacked,
	// so a slow leg does not scroll the float.
	fmt.Printf("\r\033[K%s · %ds", line, int(time.Since(p.started).Seconds()))
}

// close stops the line and erases it, so the caller's own output starts on a clean row.
func (p *progressLine) close() {
	if !p.drawing {
		return
	}
	close(p.stop)
	p.wg.Wait()
	fmt.Print("\r\033[K")
}
