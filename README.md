# dotenv

Load environment variables from `.env` files and launch a shell with them set. Supports watching for changes.

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
dotenv [options] [file]
```

### Options

- `-f <file>`: Use custom `.env` file (default: `.env`)
- `-w`: Watch for file changes and reload automatically
- `-h`: Show help

### Examples

```bash
# Load .env and start shell
dotenv

# Load specific file
dotenv -f .env.production
dotenv .env.production  # shortcut

# Watch for changes
dotenv -w
dotenv -w -f .env.development
```

## How it works

1. Reads `.env` file line by line, parsing `KEY=VALUE` pairs
2. Sets each pair as environment variable with `os.Setenv`
3. Starts your default shell (`$SHELL`) with variables loaded
4. For bash, changes prompt to `(.env) #` for visual feedback
5. In watch mode (`-w`), polls file every second and reloads on changes

## Notes

- Simple `KEY=VALUE` format only (no comments, multiline, or expansion)
- Watch mode uses 1-second polling
- Bash prompt modification only
- Variables set for spawned shell only

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.
