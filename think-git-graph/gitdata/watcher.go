package gitdata

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// RefreshNotifier is called when a git change is detected
type RefreshNotifier interface {
	NotifyRefresh()
}

// GitWatcher watches git files for changes and triggers refresh notifications
type GitWatcher struct {
	repoPath    string
	watcher     *fsnotify.Watcher
	notifier    RefreshNotifier
	debounceDur time.Duration
	debounceMu  sync.Mutex
	debounceTimer *time.Timer
	stopCh      chan struct{}
	stopped     bool
	mu          sync.RWMutex
}

// NewGitWatcher creates a new GitWatcher for the given repository path
func NewGitWatcher(repoPath string, notifier RefreshNotifier) *GitWatcher {
	return &GitWatcher{
		repoPath:    repoPath,
		notifier:    notifier,
		debounceDur: 500 * time.Millisecond,
		stopCh:      make(chan struct{}),
	}
}

// Start begins watching git files for changes
// Returns an error if file watching fails (caller should fall back to polling)
func (gw *GitWatcher) Start() error {
	gitDir := filepath.Join(gw.repoPath, ".git")
	
	// Verify .git directory exists
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("git directory not found: %s", gitDir)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	gw.watcher = watcher

	// Watch refs directory recursively
	refsDir := filepath.Join(gitDir, "refs")
	if err := gw.addDirRecursive(refsDir); err != nil {
		gw.watcher.Close()
		return fmt.Errorf("failed to watch refs directory: %w", err)
	}

	// Watch HEAD log
	headLog := filepath.Join(gitDir, "logs", "HEAD")
	if err := gw.watchFile(headLog); err != nil {
		// HEAD log might not exist yet, that's ok
		log.Printf("GitWatcher: HEAD log not watched (may not exist yet): %v", err)
	}

	// Start watching in a goroutine
	go gw.watch()

	log.Printf("GitWatcher: Started watching %s", gw.repoPath)
	return nil
}

// Stop stops the file watcher
func (gw *GitWatcher) Stop() {
	gw.mu.Lock()
	defer gw.mu.Unlock()

	if gw.stopped {
		return
	}

	gw.stopped = true
	close(gw.stopCh)

	if gw.watcher != nil {
		gw.watcher.Close()
	}

	gw.debounceMu.Lock()
	if gw.debounceTimer != nil {
		gw.debounceTimer.Stop()
	}
	gw.debounceMu.Unlock()
}

// watch is the main watching loop
func (gw *GitWatcher) watch() {
	for {
		select {
		case event, ok := <-gw.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Create == fsnotify.Create ||
				event.Op&fsnotify.Remove == fsnotify.Remove ||
				event.Op&fsnotify.Rename == fsnotify.Rename {
				gw.handleChange()
			}

		case err, ok := <-gw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("GitWatcher error: %v", err)

		case <-gw.stopCh:
			return
		}
	}
}

// handleChange debounces and triggers refresh
func (gw *GitWatcher) handleChange() {
	gw.debounceMu.Lock()
	defer gw.debounceMu.Unlock()

	// Reset timer if it exists
	if gw.debounceTimer != nil {
		gw.debounceTimer.Stop()
	}

	// Start new timer
	gw.debounceTimer = time.AfterFunc(gw.debounceDur, func() {
		if gw.notifier != nil {
			log.Printf("GitWatcher: Change detected, triggering refresh")
			gw.notifier.NotifyRefresh()
		}
	})
}

// addDirRecursive recursively adds a directory and all subdirectories to the watcher
func (gw *GitWatcher) addDirRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := gw.watcher.Add(path); err != nil {
				return fmt.Errorf("failed to watch %s: %w", path, err)
			}
		}
		return nil
	})
}

// watchFile adds a single file to the watcher (creates parent dir if needed)
func (gw *GitWatcher) watchFile(file string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(file)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create file if it doesn't exist
	if _, err := os.Stat(file); os.IsNotExist(err) {
		f, err := os.Create(file)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", file, err)
		}
		f.Close()
	}

	if err := gw.watcher.Add(file); err != nil {
		return fmt.Errorf("failed to watch file %s: %w", file, err)
	}

	return nil
}

// ChannelNotifier implements RefreshNotifier using a channel
type ChannelNotifier struct {
	ch chan struct{}
}

// NewChannelNotifier creates a new ChannelNotifier
func NewChannelNotifier() *ChannelNotifier {
	return &ChannelNotifier{
		ch: make(chan struct{}, 10),
	}
}

// NotifyRefresh sends a notification on the channel
func (cn *ChannelNotifier) NotifyRefresh() {
	select {
	case cn.ch <- struct{}{}:
	default:
		// Channel is full, drop notification
	}
}

// Chan returns the notification channel
func (cn *ChannelNotifier) Chan() <-chan struct{} {
	return cn.ch
}

// PollingFallback provides a polling-based fallback when file watching fails
type PollingFallback struct {
	repoPath    string
	notifier    RefreshNotifier
	interval    time.Duration
	stopCh      chan struct{}
	stopped     bool
	mu          sync.RWMutex
	lastModTime time.Time
}

// NewPollingFallback creates a new polling fallback
func NewPollingFallback(repoPath string, notifier RefreshNotifier, interval time.Duration) *PollingFallback {
	return &PollingFallback{
		repoPath: repoPath,
		notifier: notifier,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins polling for changes
func (pf *PollingFallback) Start() {
	go pf.poll()
}

// Stop stops the polling
func (pf *PollingFallback) Stop() {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	if pf.stopped {
		return
	}

	pf.stopped = true
	close(pf.stopCh)
}

func (pf *PollingFallback) poll() {
	ticker := time.NewTicker(pf.interval)
	defer ticker.Stop()

	// Initial check
	pf.check()

	for {
		select {
		case <-ticker.C:
			pf.check()
		case <-pf.stopCh:
			return
		}
	}
}

func (pf *PollingFallback) check() {
	gitDir := filepath.Join(pf.repoPath, ".git")
	
	// Check refs directory modification time
	refsDir := filepath.Join(gitDir, "refs")
	info, err := os.Stat(refsDir)
	if err != nil {
		return
	}

	modTime := info.ModTime()
	
	pf.mu.RLock()
	lastTime := pf.lastModTime
	pf.mu.RUnlock()

	if modTime.After(lastTime) {
		pf.mu.Lock()
		pf.lastModTime = modTime
		pf.mu.Unlock()

		if pf.notifier != nil {
			log.Printf("PollingFallback: Change detected, triggering refresh")
			pf.notifier.NotifyRefresh()
		}
	}
}

// GetRepoPath returns the absolute path to the git repository
func GetRepoPath() (string, error) {
	// Try to find .git directory by walking up
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("not a git repository")
}

// WatcherManager manages the git watcher with fallback
type WatcherManager struct {
	watcher  *GitWatcher
	fallback *PollingFallback
	notifier RefreshNotifier
	repoPath string
}

// NewWatcherManager creates a new WatcherManager
func NewWatcherManager(notifier RefreshNotifier) (*WatcherManager, error) {
	repoPath, err := GetRepoPath()
	if err != nil {
		return nil, fmt.Errorf("failed to find git repository: %w", err)
	}

	return &WatcherManager{
		notifier: notifier,
		repoPath: repoPath,
	}, nil
}

// Start starts the watcher with fallback
func (wm *WatcherManager) Start() {
	// Try file watcher first
	wm.watcher = NewGitWatcher(wm.repoPath, wm.notifier)
	if err := wm.watcher.Start(); err != nil {
		log.Printf("GitWatcher: File watching failed (%v), falling back to polling", err)
		
		// Fall back to polling
		wm.watcher = nil
		wm.fallback = NewPollingFallback(wm.repoPath, wm.notifier, 2*time.Second)
		wm.fallback.Start()
		log.Printf("GitWatcher: Using polling fallback (2s interval)")
	}
}

// Stop stops all watchers
func (wm *WatcherManager) Stop() {
	if wm.watcher != nil {
		wm.watcher.Stop()
	}
	if wm.fallback != nil {
		wm.fallback.Stop()
	}
}

// ContextNotifier implements RefreshNotifier using a context
type ContextNotifier struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewContextNotifier creates a new ContextNotifier
func NewContextNotifier() *ContextNotifier {
	ctx, cancel := context.WithCancel(context.Background())
	return &ContextNotifier{
		ctx:    ctx,
		cancel: cancel,
	}
}

// NotifyRefresh cancels the context to signal a refresh
func (cn *ContextNotifier) NotifyRefresh() {
	cn.cancel()
}

// Context returns the notification context
func (cn *ContextNotifier) Context() context.Context {
	return cn.ctx
}

// Reset creates a new context for the next refresh
func (cn *ContextNotifier) Reset() {
	cn.ctx, cn.cancel = context.WithCancel(context.Background())
}
