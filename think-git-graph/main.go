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

func main() {
	detach := flag.Bool("detach", false, "run server in background")
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

	cmd := exec.Command(exe)
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

	port, err := srv.Start()
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

	url := fmt.Sprintf("http://localhost:%d", port)

	urlFile := filepath.Join(os.TempDir(), "think-git-graph.url")
	if err := os.WriteFile(urlFile, []byte(url+"\n"), 0644); err != nil {
		log.Printf("warning: could not write URL file: %v", err)
	}

	quiet := os.Getenv("THINK_GIT_GRAPH_QUIET") == "1"

	if !quiet {
		fmt.Printf("think-git-graph listening on %s\n", url)
	}

	if !quiet {
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
