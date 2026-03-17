# monkeyrun

Production-ready, cross-platform CLI for mobile chaos (monkey) testing on **Android** and **iOS**. Uses already running emulators/simulators—no Appium, single binary.

## Features

- **CLI-first**: `devices`, `run`, `report` commands
- **Zero setup**: Works with existing ADB devices and booted iOS simulators
- **Gesture-based**: Human-like taps, double-taps, long-press, swipe, scroll, type, back
- **Crash detection**: Android (logcat) and iOS (simctl log stream) with screenshots
- **HTML report**: Summary, timeline, screenshots, logs

## Requirements

- **Android**: `adb` in PATH, device/emulator connected
- **iOS**: `xcrun` (Xcode), booted simulator, [WebDriverAgent](https://github.com/appium/WebDriverAgent) running (e.g. on `http://localhost:8100`)

## Install

### Option 1: Download prebuilt binary (recommended)

Go to the [Releases](https://github.com/ABNclearroute/monkeyrun/releases) page and download the binary for your OS:

| OS | Architecture | File |
|----|-------------|------|
| macOS | Apple Silicon (M1/M2/M3) | `monkeyrun_*_darwin_arm64.tar.gz` |
| macOS | Intel | `monkeyrun_*_darwin_amd64.tar.gz` |
| Linux | x86_64 | `monkeyrun_*_linux_amd64.tar.gz` |
| Linux | ARM64 | `monkeyrun_*_linux_arm64.tar.gz` |
| Windows | x86_64 | `monkeyrun_*_windows_amd64.zip` |

```bash
# Example: macOS Apple Silicon
curl -LO https://github.com/ABNclearroute/monkeyrun/releases/latest/download/monkeyrun_darwin_arm64.tar.gz
tar xzf monkeyrun_darwin_arm64.tar.gz
chmod +x monkeyrun
sudo mv monkeyrun /usr/local/bin/
```

### Option 2: Install with Go

```bash
go install github.com/ABNclearroute/monkeyrun@latest
```

### Option 3: Build from source

```bash
git clone https://github.com/ABNclearroute/monkeyrun.git
cd monkeyrun
make build
# binary is ./monkeyrun
```

### Cross-compile all platforms locally

```bash
make build-all
# outputs to dist/
```

## How to test

### Build

From the project root:

```bash
CGO_ENABLED=0 go build -o monkeyrun .
```

### Sanity check (no app required)

```bash
./monkeyrun devices
```

### Test on Android

- Ensure an emulator/device is **already running** and your app is **installed and in foreground**.

```bash
./monkeyrun run --platform android --app com.demo.app --events 200 --report ./report-android --verbose
```

Open the report at `./report-android/index.html`.

### Test on iOS

- Boot a simulator and ensure **WebDriverAgent is running** on `http://localhost:8100`.

```bash
./monkeyrun run --platform ios --app com.demo.app --events 200 --report ./report-ios --verbose
```

Open the report at `./report-ios/index.html`.

### Replay (optional)

```bash
./monkeyrun replay --report ./report-android --platform android --events 100
```

### Common issues

- **Android: no device found**: run `adb devices` and ensure at least one entry is `device` (not `offline`).
- **iOS: no booted simulator**: boot one, then retry `./monkeyrun devices`.
- **iOS: WDA not reachable**: verify `curl http://localhost:8100/status` works.

## Usage

```bash
# List devices
monkeyrun devices

# Run 5000 events on Android
monkeyrun run --platform android --app com.demo.app --events 5000

# Run 3000 events on iOS
monkeyrun run --platform ios --app com.demo.app --events 3000

# Custom report path and device
monkeyrun run --platform android --app com.demo.app --events 1000 --report ./out --device EMULATOR_ID

# Regenerate HTML from existing report
monkeyrun report --path report

# Replay events from a previous run
monkeyrun replay --report report --platform android --events 100
```

## Commands

| Command | Description |
|--------|-------------|
| `monkeyrun devices` | List connected Android devices and booted iOS simulators |
| `monkeyrun run` | Run monkey test (requires `--platform android\|ios`) |
| `monkeyrun report` | Generate HTML report from report dir (default: `./report`) |
| `monkeyrun replay` | Replay events from report/events.json on a connected device |

### Run flags

- `--platform` (required): `android` or `ios`
- `--app`: Package (Android) or bundle ID (iOS); device should have app running
- `--events`: Number of events (default: 1000)
- `--report`: Report output directory (default: `report`)
- `--device`: Override device/simulator ID
- `--verbose`: Verbose logging
- `--delay-min`: Min delay between actions in ms (default: 200)
- `--delay-max`: Max delay between actions in ms (default: 800)
- `--hierarchy-every`: Refresh UI hierarchy every N events (default: 1). Increase for faster runs.
- `--show-touches`: Android only: enable visual touch indicators while running
- `--stop-on-crash`: Stop execution immediately on fatal crash (default: `true`). Use `--stop-on-crash=false` to keep going.
- `--screenshot-mode`: Screenshot capture strategy (default: `balanced`). Options: `minimal`, `balanced`, `full`.
- `--screenshot-interval`: Capture a screenshot every N events in balanced/full mode (default: `25`).

## Screenshot strategy

monkeyrun uses a hybrid screenshot capture strategy that balances performance and visual debugging. Instead of capturing every event, screenshots are taken intelligently.

| Mode | Captures when | Best for |
|------|---------------|----------|
| `minimal` | Crash only | CI/CD pipelines, maximum speed |
| `balanced` (default) | Every N events, UI changes, crashes | General use |
| `full` | Every event | Detailed debugging |

**UI change detection**: In `balanced` mode, monkeyrun hashes the current UI hierarchy (SHA-256) and compares it with the previous hash. A screenshot is taken whenever the UI visually changes, even outside the interval.

Screenshots are captured asynchronously via a worker pool so the monkey test is never blocked. Crash screenshots are always captured synchronously to guarantee availability.

```bash
# Minimal screenshots (fast, small reports)
monkeyrun run --platform android --app com.demo.app --events 5000 --screenshot-mode minimal

# Balanced with custom interval
monkeyrun run --platform android --app com.demo.app --events 5000 --screenshot-interval 50

# Full screenshots for debugging
monkeyrun run --platform android --app com.demo.app --events 200 --screenshot-mode full
```

## Crash handling

monkeyrun classifies crashes into two severity levels:

| Severity | Keywords (Android) | Keywords (iOS) | Behavior |
|----------|-------------------|----------------|----------|
| **Fatal** | `FATAL EXCEPTION`, `SIGSEGV`, `Fatal signal`, `ANR in` | `SIGABRT`, `SIGSEGV`, `Terminating app`, `Exception Type`, `fatal error` | Stops execution (with `--stop-on-crash`), screenshot + logs saved |
| **Minor** | `AndroidRuntime`, `Force finishing`, `has died` | `Assertion failed`, `crash` | Logged to report, execution continues |

The engine also stops if **5 consecutive UI hierarchy errors** occur (meaning the app likely left the foreground or is unresponsive).

## Report layout

```
report/
  index.html      # Summary, timeline, crashes, screenshots, logs
  events.json     # Event log (includes x/y coordinates and screenshot flags)
  screenshots/    # Event and crash screenshots (event_N.png, crash_N.png)
  logs/           # crash.log
```

## Architecture

- **CLI** (Cobra) → **Engine** → **Device** interface → **Android** (ADB) / **iOS** (WebDriverAgent + simctl)
- UI hierarchy: Android via `uiautomator dump`, iOS via WDA `/source`
- Actions: weighted random (tap 40%, swipe 20%, etc.) with element-aware choices

## Releasing a new version

Tag and push — GitHub Actions will build and publish binaries automatically:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Binaries for macOS, Linux, and Windows (amd64 + arm64) will appear on the [Releases](https://github.com/ABNclearroute/monkeyrun/releases) page within a few minutes.

## License

MIT
