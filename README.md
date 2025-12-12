# dotenv

Load environment variables from `.env` files and launch a shell or command with them set. Supports watching for changes.

## Installation

### Build from source

```bash
git clone https://github.com/yshngg/dotenv.git
cd dotenv
go build -o dotenv main.go
sudo install -m 755 dotenv /usr/local/bin/dotenv
```

### Using go install

```bash
go install github.com/yshngg/dotenv@latest
```

## Usage

```bash
dotenv [-f <file>] [-w] [-h] [-- <command>]
```

### Options

- `-f <file>`: Use custom `.env` file (default: `.env`)
- `-w`: Watch for file changes and reload automatically
- `--`: Separator to specify custom command to run (default: `$SHELL`)
- `-h`: Show help

**Note:** Custom commands must be preceded by `--` separator. Positional file arguments are not supported.

### Examples

```bash
# Load .env and start default shell
dotenv

# Load specific .env file
dotenv -f .env.production

# Watch .env for changes and restart shell
dotenv -w
dotenv -w -f .env.development

# Run custom command with environment variables
dotenv -- python script.py
dotenv -- npm start
dotenv -f .env.test -- go test ./...

# Watch mode with custom command
dotenv -w -- python manage.py runserver
dotenv -w -f .env.local -- ./myapp
```

## How it works

1. Reads `.env` file line by line, parsing `KEY=VALUE` pairs (invalid lines are skipped)
2. Appends variables to the environment of the spawned command
3. Starts your default shell (`$SHELL`) or custom command specified after `--`
4. For bash shells (command ends with "bash"), changes prompt to `(.env) #` for visual feedback
5. Creates a process group for proper signal delivery to child processes
6. In watch mode (`-w`), polls file every 100ms and reloads on changes
7. On file change, kills the current process group and restarts the command with updated environment

## Notes

- Simple `KEY=VALUE` format only (no comments, multiline, or expansion); whitespace around `=` is trimmed
- Invalid lines are logged and skipped
- Environment file must exist; tool exits with error if file not found
- Watch mode uses 100ms polling
- Bash prompt modification only when command ends with "bash"
- Variables set for spawned command only (not current process)
- Custom commands can be specified after `--` separator
- Process groups ensure proper signal delivery to child processes (uses `pkill -P` for termination)
- Informational messages logged to stderr (file changes, process termination, restarts)

## Testing

The `test/` directory contains example programs for testing dotenv functionality:

- `test/counter/` - Simple counter that prints incrementing numbers (useful for testing watch mode)
- `test/server/` - HTTP server that displays environment variables

Example usage with test programs:

```bash
# Run counter with environment variables
dotenv -- go run test/counter/main.go

# Run server with environment variables in watch mode
dotenv -w -- go run test/server/main.go
```

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.
