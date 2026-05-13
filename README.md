# Rockpload

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.26.1-00ADD8?logo=go)](https://go.dev/)
[![Discord](https://img.shields.io/discord/1503472270478413864?color=7289da&label=discord&logo=discord&logoColor=white)](https://discord.gg/bFw6mcVXSW)

<p align="center">
   <img src="app/assets/logo.png" alt="Rockpload logo" width="360" />
</p>


A desktop companion app for Rocket League that watches your replay history and automatically uploads new replays to any website!

## Features

- **Automatic Upload**: Monitors your Rocket League replay directory and uploads new replays in real time
- **Epic Games Authentication**: Seamless authentication via Epic Games Launcher or custom OAuth
- **Multi Account Upload**: Allow to upload for multiple account at the same time
- **Multi Website Upload**: Allow to upload for multiple website at the same time ([Rocky](https://lexore.ca/rocky) and [Ballchasing](https://ballchasing.com) are preconfigured, but you can add as many as you want)
- **System Tray Integration**: Runs quietly in the background with system tray controls
- **Auto-start Support**: Launch rockpload automatically on system startup
- **Cross-platform**: Supports Linux and Windows (macOS is not supported yet)
- **Console Platform**: Supports Epic Games Store, Steam, and console (PlayStation, XBox, Nintendo) as long as the program run on a PC

## Supported Platforms

- **Linux** (amd64)
- **Windows** (amd64)
- **macOS**: Not supported yet

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

## Usage

Run the application:
```bash
./rockpload  # Linux
rockpload.exe  # Windows
```

Setup:
1. Start the program on the PC
2. Connect to your Epic account (if you are on another console/store then Epic, be sure that your account is linked to Epic)
3. All the settings will be up right (Account Management, Storage Management and App Settings). Its there that you activate the auto upload
4. If you wanna upload to Ballchasing, go to Storage Management and be sure that the Token is there and that the "Send Replay" is checked
5. For not being disconnect while playing the game, be sure to have an account in the "Unused Account" section. This account is used to check for Online Status of all others account you have before trying to upload

**Local State:**
- Config: `~/.cache/rockpload/` (Linux) or `%LOCALAPPDATA%\rockpload\` (Windows)
- Cache: `~/.cache/rockpload/`

**Note:** If you run the program on multiple devices, each device will need to authenticate separately. Only one device per account can maintain a valid token at a time.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on:
- Setting up your development environment
- Code style and formatting requirements
- Pull request workflow

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

Copyright 2026 LEX0RE
