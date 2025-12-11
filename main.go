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

type option struct {
	filepath string
	cmd      []string
	watch    bool
	help     bool
}

func (o *option) validate() error {
	if _, err := os.Stat(o.filepath); err != nil {
		return err
	}
	if len(o.cmd) == 0 {
		return fmt.Errorf("invalid command")
	}
	return nil
}

func parseArgs(args []string) (option, error) {
	shell := os.Getenv("SHELL")
	opt := option{
		filepath: DefaultDotEnvFilepath,
		cmd:      []string{shell},
		watch:    false,
		help:     false,
	}

	for len(args) > 0 {
		switch args[0] {
		case "-h":
			opt.help = true
			return opt, nil
		case "-f":
			opt.filepath = args[1]
			args = args[2:]
		case "-w":
			opt.watch = true
			args = args[1:]
		case "--":
			opt.cmd = args[1:]
			args = []string{}
		default:
			return opt, fmt.Errorf("invalid argument: %s", args[0])
		}
	}

	return opt, nil
}

func main() {
	opt, err := parseArgs(os.Args[1:])
	if err != nil {
		log.Fatalf("Error parsing arguments: %v", err)
		return
	}

	if opt.help {
		printUsage()
		return
	}

	env, err := getEnviron(opt.filepath)
	if err != nil {
		log.Fatalf("Error getting Environ: %v", err)
		return
	}
	if strings.HasSuffix(opt.cmd[0], string(ShellTypeBash)) {
		os.Setenv("PS1", "(.env) # ")
	}
	cmd, err := runCommand(opt.cmd[0], opt.cmd[1:], env)
	if err != nil {
		log.Fatalf("Error running command: %v", err)
		return
	}
	waitChan := make(chan *exec.Cmd, 1)
	waitChan <- cmd

	if !opt.watch {
		close(waitChan)
	} else {
		killChan := make(chan *exec.Cmd)

		go func() {
			for cmd := range killChan {
				if cmd == nil {
					continue
				}
				err = cmd.Process.Kill()
				if err != nil {
					log.Printf("Error killing process: %v", err)
				}
			}
		}()

		go func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			watchChan, err := watchFile(ctx, opt.filepath)
			if err != nil {
				log.Fatalf("Error watching file: %v", err)
				return
			}
			for {
				select {
				case <-watchChan:
					if len(waitChan) != 0 {
						killChan <- <-waitChan
					}
					env, err = getEnviron(opt.filepath)
					if err != nil {
						log.Fatalf("Error getting Environ: %v", err)
						return
					}
					if strings.HasSuffix(opt.cmd[0], string(ShellTypeBash)) {
						os.Setenv("PS1", "(.env) # ")
					}

					cmd, err := runCommand(opt.cmd[0], opt.cmd[1:], env)
					if err != nil {
						log.Fatalf("Error running command: %v", err)
						return
					}
					waitChan <- cmd
				case <-ctx.Done():
					close(waitChan)
					return
				}
			}
		}()
	}

	for cmd := range waitChan {
		if err = cmd.Wait(); err != nil {
			log.Fatalf("Error waiting for command: %v", err)
		}
	}
}

func runCommand(name string, args []string, env []string) (*exec.Cmd, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), env...)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("running %s %s, err: %w", cmd, args, err)
	}
	return cmd, nil
}

func getEnviron(filepath string) ([]string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("open file: %s, err: %w", filepath, err)
	}
	defer f.Close()

	env := make([]string, 0)
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
		env = append(env, key+"="+value)
	}
	return env, nil
}

func watchFile(ctx context.Context, filepath string) (<-chan struct{}, error) {
	ch := make(chan struct{})
	former, err := os.Stat(filepath)
	if err != nil {
		return nil, fmt.Errorf("stat file, err: %w", err)
	}

	go func() {
		defer close(ch)

		ticker := time.NewTicker(time.Millisecond * 100)
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
