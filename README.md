# FlibustaHub

**English** · [Русский](README.ru.md)

A local catalog for your book library: search, personal ratings, an OPDS server for e-readers and AI recommendations. Desktop application for Windows and Linux.

> **Status:** in development. This README is a skeleton; the full text is written closer to the first release.

## Features

- Imports a catalog from an `.inpx` dump (around 700,000 books) into a local SQLite database.
- Fast search over titles, authors and series through FTS5.
- Personal ratings and notes that survive the monthly re-import.
- Reads and extracts `fb2` files straight from `zip` archives, without unpacking them to disk.
- Built-in OPDS server for e-ink readers on your home network.
- AI recommendations based on your own ratings (Gemini, OpenAI, Ollama).
- Works without an internet connection.

## Requirements

- Windows 10 or newer (WebView2), or Linux with WebKitGTK.
- A library dump: an `.inpx` file and the `zip` archives with the books.

## Installation

TODO: link to the releases page.

The binaries are not code-signed, so Windows SmartScreen may warn you on first launch. That is expected.

## First run

TODO: the wizard — choosing the library folder, locating the `.inpx` file, running the import.

## Connecting an e-reader over OPDS

TODO: enabling public access, choosing the network interface, the address and the QR code.

## AI recommendations

TODO: supported providers, where to get an API key, where the key is stored (OS credential storage) and why it never leaves the machine.

## Where the data is stored

TODO: the data directory, the database, the cover cache, logs and backups. Personal ratings and notes are stored separately from the library folder and stay available even when that folder is not.

## Building from source

Requires Go 1.25+, Node.js 22, [Task](https://taskfile.dev), and the Wails v2 CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0`). On Linux also install `build-essential`, `pkg-config`, `libgtk-3-dev`, and `libwebkit2gtk-4.1-dev`.

```bash
task build
```

The binary is written to `build/bin/FlibustaHub.exe` on Windows and `build/bin/FlibustaHub` on Linux. That directory is gitignored. Use this release build (not `wails dev`) for local acceptance and performance checks.

## Architecture

TODO: the layer diagram (Wails bindings and the HTTP server → shared application services → repositories → SQLite and the file system).

## Documentation

- Change history: `CHANGELOG.md`.
- How to contribute: `CONTRIBUTING.md`.

## License

MIT — see `LICENSE`.

## Disclaimer

This application is a catalog for **the user's own local files**. It does not contain, distribute or download books, and it contains no links to library dump sources. Responsibility for the legality of the files used lies with the user.
