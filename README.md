# Rockpload

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.26.1-00ADD8?logo=go)](https://go.dev/)
[![GitHub Workflow Status](https://img.shields.io/github/workflow/status/LEX0RE/rockpload/CI)](https://github.com/LEX0RE/rockpload/actions)

<p align="center">
   <img src="app/assets/logo.png" alt="Rockpload logo" width="360" />
</p>


A desktop companion app for [Rocky](https://lexore.ca/rocky) that watches your Rocket League replay folder and automatically uploads new replays to the Rocky stats tracker.

## Features

- **Automatic Upload**: Monitors your Rocket League replay directory and uploads new replays in real time
- **Epic Games Authentication**: Seamless authentication via Epic Games Launcher or custom OAuth
- **System Tray Integration**: Runs quietly in the background with system tray controls
- **Auto-start Support**: Launch rockpload automatically on system startup
- **Cross-platform**: Supports Linux and Windows (macOS is not supported)

## Supported Platforms

- **Linux** (amd64)
- **Windows** (amd64)
- **macOS**: Not supported

## Installation

### Download Pre-built Binary

Download the latest release from the [GitHub Releases](https://github.com/LEX0RE/rockpload/releases) page.

### Build from Source

**Prerequisites:**
- Go 1.26.1 or later
- CGO toolchain (GCC on Linux, MinGW on Windows for cross-compilation)

**Build:**
```bash
./build.sh
```

The build script will produce binaries for Linux and Windows in the current directory.

## Configuration

1. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

2. Configure authentication mode in `.env`:

   **Launcher Mode (Default):**
   ```bash
   EPIC_AUTH_MODE=launcher
   ```
   Uses Epic Games Launcher's built-in Rocket League authentication. This is the recommended mode for most users.

   **Custom OAuth Mode:**
   ```bash
   EPIC_AUTH_MODE=custom
   EPIC_CLIENT_ID=your_client_id
   EPIC_CLIENT_SECRET=your_client_secret
   ```
   Requires your own Epic Games OAuth application credentials. Note: Standard Epic OAuth clients cannot create the Rocket League exchange code required by this app.

## Usage

Run the application:
```bash
./rockpload  # Linux
rockpload.exe  # Windows
```

The app will:
1. Authenticate with Epic Games
2. Monitor your Rocket League replay folder
3. Automatically upload new replays to https://lexore.ca/rocky/api
4. Run in the system tray

**Local State:**
- Rocket League tokens: `~/.cache/rockpload/.rltoken` (Linux) or `%LOCALAPPDATA%\rockpload\.rltoken` (Windows)
- Upload cache: `~/.cache/rockpload/.uploaded`
- Application lock: `~/.cache/rockpload/rockpload.lock`

**Note:** If you run the program on multiple devices, each device will need to authenticate separately. Only one device per account can maintain a valid token at a time.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on:
- Setting up your development environment
- Code style and formatting requirements
- Pull request workflow

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

Copyright 2026 LEX0RE
