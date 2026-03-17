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

```bash
go build -o monkeyrun .
# or CGO_ENABLED=0 for static binary
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

## Report layout

```
report/
  index.html      # Summary, timeline, crashes, screenshots, logs
  events.json     # Event log
  screenshots/    # Crash screenshots
  logs/           # crash.log
```

## Architecture

- **CLI** (Cobra) → **Engine** → **Device** interface → **Android** (ADB) / **iOS** (WebDriverAgent + simctl)
- UI hierarchy: Android via `uiautomator dump`, iOS via WDA `/source`
- Actions: weighted random (tap 40%, swipe 20%, etc.) with element-aware choices

## License

MIT
