package internal

import (
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/fsnotify/fsnotify"
	ignore "github.com/sabhiram/go-gitignore"
)

const watchDebounce = 200 * time.Millisecond

var watchStatusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))

// Watch runs the initial test pass then enters a debounced rerun loop keyed on .go file changes. `r` force rerun, `q` or ctrl+c quits.
func Watch(args []string) {
	scope := "./..."
	if len(args) > 0 {
		scope = strings.Join(args, " ")
	}
	fmt.Printf("[watch] watching %s\n", scope)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Fprintf(os.Stderr, "watcher: %v\n", err)
		return
	}
	defer watcher.Close()

	matcher := loadIgnore(".")
	if err := watchRecursive(watcher, ".", matcher, nil); err != nil {
		fmt.Fprintf(os.Stderr, "watch setup: %v\n", err)
	}

	// Pre-render ui strings to avoid allocations during the run loop.
	status := watchStatusStyle.Render("[watch] idle · press r to rerun · q to quit")
	clear := ansi.CursorHomePosition + ansi.EraseEntireScreen + ansi.EraseEntireDisplay

	Execute(args)
	fmt.Println(status)

	fd := os.Stdin.Fd()
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "raw mode: %v\n", err)
		return
	}
	defer term.Restore(fd, oldState)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	keyCh := make(chan byte, 1)
	go readKeys(keyCh)

	rerun := func(trigger string) {
		_ = term.Restore(fd, oldState)
		fmt.Printf("%s[watch] trigger: %s\n\n", clear, trigger)
		Execute(args)
		fmt.Println(status)
		if _, err := term.MakeRaw(fd); err != nil {
			fmt.Fprintf(os.Stderr, "raw mode: %v\n", err)
		}
	}

	var (
		timer       *time.Timer
		timerC      <-chan time.Time
		lastTrigger string
	)

	schedule := func(trigger string) {
		lastTrigger = trigger
		if timer == nil {
			timer = time.NewTimer(watchDebounce)
		} else {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(watchDebounce)
		}
		timerC = timer.C
	}

	for {
		select {
		case ev, ok := <-watcher.Events:
			if !ok {
				return
			}
			if ev.Has(fsnotify.Create) {
				if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
					// fsnotify only sees changes from here on; a directory
					// that arrived with files already inside it needs a manual nudge.
					var foundGoFile string
					err := watchRecursive(watcher, ev.Name, matcher, func(p string) {
						if foundGoFile == "" {
							foundGoFile = p
						}
					})
					if err != nil {
						fmt.Fprintf(os.Stderr, "watch add: %v\n", err)
					}
					if foundGoFile != "" {
						schedule(filepath.Base(foundGoFile))
					}
				}
				continue
			}
			if !strings.HasSuffix(ev.Name, ".go") {
				continue
			}
			schedule(filepath.Base(ev.Name))

		case <-timerC:
			timer, timerC = nil, nil
			rerun(lastTrigger)

		case b, ok := <-keyCh:
			if !ok {
				return
			}
			switch b {
			case 'r', 'R':
				if timer != nil {
					timer.Stop()
					timer, timerC = nil, nil
				}
				rerun("manual")
			case 'q', 'Q', 0x03: // Ctrl+C
				return
			}

		case <-sigCh:
			return

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "watch error: %v\n", err)
		}
	}
}

// readKeys streams bytes from stdin until it hits EOF or a read error.
func readKeys(out chan<- byte) {
	defer close(out)
	b := make([]byte, 1)
	for {
		if _, err := os.Stdin.Read(b); err != nil {
			return
		}
		out <- b[0]
	}
}

func loadIgnore(root string) *ignore.GitIgnore {
	patterns := []string{".git/"}
	path := filepath.Join(root, ".gitignore")
	if data, err := os.ReadFile(path); err == nil {
		for line := range strings.SplitSeq(string(data), "\n") {
			if line != "" {
				patterns = append(patterns, line)
			}
		}
	}
	return ignore.CompileIgnoreLines(patterns...)
}

// isIgnored reports whether path matches the .gitignore patterns. isDir
// must reflect whether path is a directory, since gitignore patterns can
// be directory only.
func isIgnored(path string, isDir bool, matcher *ignore.GitIgnore) bool {
	if matcher == nil {
		return false
	}
	if isDir {
		// Ensure trailing slash for directory matching
		if !strings.HasSuffix(path, string(filepath.Separator)) {
			path += string(filepath.Separator)
		}
	}
	return matcher.MatchesPath(path)
}

// watchRecursive adds directories to the watcher. If onGoFile is provided,
// it will call it with the first .go file it encounters.
func watchRecursive(w *fsnotify.Watcher, root string, matcher *ignore.GitIgnore, onGoFile func(string)) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if isIgnored(path, true, matcher) {
				return filepath.SkipDir
			}
			return w.Add(path)
		}
		if onGoFile != nil && strings.HasSuffix(path, ".go") && !isIgnored(path, false, matcher) {
			onGoFile(path)
		}
		return nil
	})
}
