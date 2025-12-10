package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

const DefaultDotEnvFilepath = ".env"

type ShellType string

const (
	ShellTypeBash ShellType = "bash"
)

func printUsage() {
	fmt.Println("Usage: dotenv [-f <file>] [-w]")
}

func parseArgs(args []string) (string, bool, bool) {
	var (
		filepath string = DefaultDotEnvFilepath
		watch    bool   = false
	)
	for len(args) > 0 {
		switch args[0] {
		case "-h":
			return "", false, true
		case "-f":
			filepath = args[1]
			args = args[2:]
		case "-w":
			watch = true
			args = args[1:]
		default:
			if _, err := os.Stat(filepath); err != nil {
				return "", false, true
			}
			filepath = args[0]
			args = args[1:]
		}
	}

	return filepath, watch, false
}

func main() {
	filepath, watch, help := parseArgs(os.Args[1:])
	if help {
		printUsage()
		return
	}

	if watch {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		watchChan, err := watchFile(ctx, filepath)
		if err != nil {
			log.Fatalf("Error watching file: %v", err)
			return
		}

		go func() {
			for {
				select {
				case <-watchChan:
					setEnviron(filepath)
				case <-ctx.Done():
					return
				}
			}
		}()
		runShell()
		return
	}
	setEnviron(filepath)
	runShell()
}

func runShell() {
	shell := os.Getenv("SHELL")
	if strings.HasSuffix(shell, string(ShellTypeBash)) {
		os.Setenv("PS1", "(.env) # ")
	}

	cmd := exec.Command(shell)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatalf("Error running %s: %v", shell, err)
	}
}

func setEnviron(filepath string) error {
	f, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("open file: %s, err: %w", filepath, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		keyValue := strings.Split(string(line), "=")
		if len(keyValue) != 2 {
			log.Fatalf("Invalid line format %s", string(line))
			continue
		}
		key := strings.TrimSpace(keyValue[0])
		value := strings.TrimSpace(keyValue[1])
		if err := os.Setenv(key, value); err != nil {
			log.Fatalf("Error setting environment variable %s=%s: %v", key, value, err)
		}
	}
	return nil
}

func watchFile(ctx context.Context, filepath string) (<-chan struct{}, error) {
	ch := make(chan struct{})
	former, err := os.Stat(filepath)
	if err != nil {
		return nil, fmt.Errorf("stat file, err: %w", err)
	}

	go func() {
		defer close(ch)

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				latter, err := os.Stat(filepath)
				if err != nil {
					log.Fatalf("Error getting file info: %v", err)
					continue
				}
				if latter.ModTime().After(former.ModTime()) {
					log.Printf("File %s changed", filepath)
					ch <- struct{}{}
					former = latter
				}
			}
		}
	}()

	return ch, nil
}
