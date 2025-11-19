# Editor Bot (emailbot)

AI-themed combo editing tool built with Go. Users import `email:password` combos, and Editor Bot automatically applies popular modification patterns, deduplicates, randomizes output, and logs the AI "training" progress across sessions.

## Features
- **AI Edit Method UI**: dynamic banner and messaging that encourages importing larger datasets for a "smarter" bot.
- **Mass Data Handling**: buffered streaming, randomized output, and automatic duplicate removal.
- **Encrypted Usage Counter**: cumulative import count stored via AES-256-GCM in `data/usage.enc`, displayed on launch.
- **PowerShell GUI Input**: invokes Windows file dialog to select combo files.
- **Tested**: includes `main_test.go` for usage counter persistence.

## Getting Started
```bash
go run main.go
```
The bot prompts for a combo file, processes each `email:password` line, generates three AI-styled variations, removes duplicates, shuffles the data, and writes to `output/<timestamp>/output.txt`.

## Build & Test
```bash
go build -o editor-bot.exe

go test ./...
```

## Notes
- Output and encrypted usage data live in `output/` and `data/` respectively.
- Deleting `data/usage.enc` resets the cumulative AI training counter.
