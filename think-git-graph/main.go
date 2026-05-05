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
	"runtime"
	"syscall"
	"time"

	"github.com/dalpat/git-tools/think-git-graph/gitdata"
	"github.com/dalpat/git-tools/think-git-graph/server"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	detach := flag.Bool("detach", false, "run server in background (not yet implemented)")
	flag.Parse()

	if *detach {
		log.Println("--detach flag is not yet implemented; running in foreground")
	}

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
	fmt.Printf("think-git-graph listening on %s\n", url)

	if err := openBrowser(url); err != nil {
		log.Printf("could not open browser: %v", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nshutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Stop(); err != nil {
		log.Printf("error shutting down: %v", err)
	}
	// Wait for shutdown to complete
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
