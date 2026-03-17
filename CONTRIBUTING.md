# Contributing to monkeyrun

Thank you for your interest in contributing to monkeyrun! This guide will help you get started.

## Getting Started

### Prerequisites

- **Go 1.21+** — [install](https://go.dev/dl/)
- **Android** (for Android testing): `adb` in PATH, emulator or device connected
- **iOS** (for iOS testing): Xcode with `xcrun`, booted simulator, [WebDriverAgent](https://github.com/appium/WebDriverAgent) running

### Clone and build

```bash
git clone https://github.com/ABNclearroute/monkeyrun.git
cd monkeyrun
make build
```

### Run locally

```bash
# List connected devices
./monkeyrun devices

# Run a quick test
./monkeyrun run --platform android --app com.demo.app --events 100 --verbose
```

### Run tests

```bash
make test
```

### Lint and format

```bash
make fmt
make lint
```

## Project Structure

```
monkeyrun/
├── cmd/            CLI commands (Cobra): devices, run, report, replay
├── device/         Device interface + platform adapters (Android/ADB, iOS/WDA)
│   ├── device.go           Core interface (Gesturer, Inspector, Logger)
│   ├── android.go          Android adapter (ADB shell commands)
│   ├── ios.go              iOS adapter (WebDriverAgent + xcrun simctl)
│   ├── android_parser.go   Parse uiautomator XML into UIElements
│   ├── ios_parser.go       Parse WDA /source XML/JSON into UIElements
│   ├── factory.go          Auto-detect and create devices
│   └── detect.go           Device discovery helpers
├── engine/         Monkey engine: action selection, execution, screenshot strategy
│   ├── monkey_engine.go    Core event loop with UI hierarchy caching
│   ├── actions.go          Action types and structs
│   ├── executor.go         Execute actions with human-like delays
│   ├── screenshot.go       Hybrid screenshot capture with async worker pool
│   └── replay.go           Replay events from JSON
├── crash/          Crash detection from log streams
│   └── detector.go         Fatal/minor keyword matching
├── report/         HTML report generation and event logging
│   └── report.go           Playwright-style dark-themed report
├── main.go         Entry point
├── Makefile        Build, test, lint targets
└── .github/        CI workflows, issue/PR templates
```

## How to Contribute

### 1. Fork the repo

Click **Fork** on GitHub, then clone your fork:

```bash
git clone https://github.com/YOUR_USERNAME/monkeyrun.git
cd monkeyrun
```

### 2. Create a feature branch

```bash
git checkout -b feat/my-new-feature
```

### 3. Make your changes

- Write code
- Add or update tests
- Run `make test` and `make lint` before committing

### 4. Commit with a clear message

Follow the [commit message convention](#commit-message-convention) below.

### 5. Push and open a PR

```bash
git push origin feat/my-new-feature
```

Then open a Pull Request on GitHub against `main`. Fill in the PR template.

## Coding Guidelines

- **Follow Go conventions** — use `gofmt`, pass `go vet`, and keep code idiomatic.
- **Small, focused functions** — each function should do one thing well.
- **Meaningful names** — variables, functions, and types should be self-documenting.
- **Comment non-obvious logic** — don't restate what the code does; explain *why*.
- **No unused code** — remove dead imports, variables, and functions.
- **Handle errors** — never silently discard errors unless intentional (use `_ =` with a comment).
- **Keep dependencies minimal** — monkeyrun aims to be a lightweight single binary.

## Commit Message Convention

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>: <short description>
```

| Type       | When to use                                |
|------------|--------------------------------------------|
| `feat`     | New feature or capability                  |
| `fix`      | Bug fix                                    |
| `docs`     | Documentation only                         |
| `refactor` | Code change that neither fixes nor adds    |
| `test`     | Adding or updating tests                   |
| `chore`    | Build, CI, tooling changes                 |

Examples:

```
feat: add swipe action for iOS
fix: crash detection missing ANR keyword
docs: update README install section
test: add unit tests for Android UI parser
refactor: extract screenshot worker pool
```

## Reporting Issues

- Use the [bug report](.github/ISSUE_TEMPLATE/bug_report.md) template for bugs.
- Use the [feature request](.github/ISSUE_TEMPLATE/feature_request.md) template for ideas.
- Check existing issues before creating a new one.

## Code of Conduct

Be respectful, constructive, and inclusive. We're all here to build better tools.
