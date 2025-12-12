package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
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

func parseArgs(args []string) (*option, error) {
	shell := os.Getenv("SHELL")
	opt := &option{
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
			return nil, fmt.Errorf("invalid argument: %s", args[0])
		}
	}

	return opt, nil
}

func main() {
	opt, err := parseArgs(os.Args[1:])
	if err != nil {
		log.Fatalf("Error parsing arguments: %v", err)
	}

	err = opt.validate()
	if err != nil {
		log.Printf("Error validating option: %v", err)
	}

	if err != nil || opt.help {
		printUsage()
		return
	}

	log.Printf("Using env file: %s", opt.filepath)

	env, err := getEnviron(opt.filepath)
	if err != nil {
		log.Fatalf("Error getting Environ: %v", err)
	}
	if strings.HasSuffix(opt.cmd[0], string(ShellTypeBash)) {
		env = append(env, "PS1=(.env) # ")
	}
	cmd, err := runCommand(opt.cmd[0], opt.cmd[1:], env)
	if err != nil {
		log.Fatalf("Error running command: %v", err)
	}
	waitChan := make(chan *exec.Cmd, 1)
	waitChan <- cmd

	if !opt.watch {
		close(waitChan)
	} else {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func(ctx context.Context, cmd *exec.Cmd) {
			defer close(waitChan)

			watchChan, err := watchFile(ctx, opt.filepath)
			if err != nil {
				log.Fatalf("Error watching file: %v", err)
			}
			for {
				if cmd != nil && cmd.ProcessState != nil {
					return
				}
				select {
				case <-watchChan:
					if cmd != nil {
						if err = exec.Command("pkill", "-P", fmt.Sprintf("%d", cmd.Process.Pid)).Run(); err != nil {
							log.Printf("Error killing process %d: %v", cmd.Process.Pid, err)
						} else {
							log.Printf("Killed process: %d", cmd.Process.Pid)
						}
					}
					env, err = getEnviron(opt.filepath)
					if err != nil {
						log.Fatalf("Error getting Environ: %v", err)
					}
					if strings.HasSuffix(opt.cmd[0], string(ShellTypeBash)) {
						env = append(env, "PS1=(.env) # ")
					}

					cmd, err = runCommand(opt.cmd[0], opt.cmd[1:], env)
					if err != nil {
						log.Fatalf("Error running command: %v", err)
					}
					log.Printf("Restarted command: %s %s", opt.cmd[0], strings.Join(opt.cmd[1:], " "))
					waitChan <- cmd
				case <-ctx.Done():
					return
				default: // no blocking
				}
			}
		}(ctx, cmd)
	}

	for cmd := range waitChan {
		if err = cmd.Wait(); err != nil {
			log.Printf("Error waiting for command: %v", err)
		}
	}
}

func runCommand(name string, args []string, env []string) (*exec.Cmd, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), env...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: os.Getpid()}

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
			log.Printf("Invalid line format %s", string(line))
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
					log.Printf("Error getting file info: %v", err)
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
