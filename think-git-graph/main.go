package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/dalpat/git-tools/think-git-graph/gitdata"
	"github.com/dalpat/git-tools/think-git-graph/server"
)

//go:embed static/*
var staticFiles embed.FS

var (
	detach = flag.Bool("detach", false, "run server in background")
	port   = flag.String("port", "", "port to listen on (default: random)")
	host   = flag.String("host", "127.0.0.1", "host to bind to")
	noOpen = flag.Bool("no-open", false, "do not open browser automatically")
)

func main() {
	flag.Parse()

	if *detach {
		runDetached()
		return
	}

	runServer()
}

func runDetached() {
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("failed to get executable path: %v", err)
	}

	urlFile := filepath.Join(os.TempDir(), "think-git-graph.url")
	os.Remove(urlFile)

	// Pass through non-detach flags to the background process
	var args []string
	for _, arg := range os.Args[1:] {
		if arg != "--detach" && arg != "-detach" {
			args = append(args, arg)
		}
	}
	cmd := exec.Command(exe, args...)
	cmd.Env = append(os.Environ(), "THINK_GIT_GRAPH_QUIET=1")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("failed to start detached process: %v", err)
	}

	pidFile := filepath.Join(os.TempDir(), "think-git-graph.pid")
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0644); err != nil {
		log.Printf("warning: could not write PID file: %v", err)
	}

	var url string
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		if data, err := os.ReadFile(urlFile); err == nil {
			url = strings.TrimSpace(string(data))
			break
		}
	}

	if url != "" {
		fmt.Printf("think-git-graph listening on %s\n", url)
	}
	fmt.Printf("think-git-graph started in background (PID %d)\n", cmd.Process.Pid)
	fmt.Printf("PID file: %s\n", pidFile)
	fmt.Println("Stop with: kill $(cat " + pidFile + ")")
}

func runServer() {
	gd := gitdata.NewReal()

	srv, err := server.New(gd, staticFiles)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	addr := fmt.Sprintf("%s:0", *host)
	if *port != "" {
		addr = fmt.Sprintf("%s:%s", *host, *port)
	}
	portNum, err := srv.Start(addr)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

	url := fmt.Sprintf("http://%s:%d", *host, portNum)

	urlFile := filepath.Join(os.TempDir(), "think-git-graph.url")
	if err := os.WriteFile(urlFile, []byte(url+"\n"), 0644); err != nil {
		log.Printf("warning: could not write URL file: %v", err)
	}

	quiet := os.Getenv("THINK_GIT_GRAPH_QUIET") == "1"

	if !quiet {
		fmt.Printf("think-git-graph listening on %s\n", url)
	}

	// Initialize GitWatcher with server as notifier
	var watcherManager *gitdata.WatcherManager
	if _, err := gitdata.GetRepoPath(); err == nil {
		watcherManager, err = gitdata.NewWatcherManager(srv)
		if err != nil {
			log.Printf("warning: could not create git watcher: %v", err)
			log.Printf("continuing with manual refresh only")
		} else {
			watcherManager.Start()
			if !quiet {
				fmt.Println("Git file watcher: started")
			}
		}
	} else {
		log.Printf("warning: not in a git repository: %v", err)
		log.Printf("continuing without auto-refresh")
	}

	if !quiet && !*noOpen {
		if err := openBrowser(url); err != nil {
			log.Printf("could not open browser: %v", err)
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	if !quiet {
		fmt.Println("\nshutting down...")
	}

	// Stop the watcher
	if watcherManager != nil {
		watcherManager.Stop()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Stop(); err != nil {
		log.Printf("error shutting down: %v", err)
	}
	<-ctx.Done()
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}
