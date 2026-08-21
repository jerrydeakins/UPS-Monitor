# UPS Monitor

Windows application for monitoring an APCUPSD UPS over the APCUPSD Network Information Server (NIS).

## Features

- Monitors UPS status through APCUPSD.
- Shows UPS state, battery level, load and remaining runtime.
- System tray integration.
- Pause/resume monitoring.
- Automatic shutdown after configurable time on battery.
- Warning window with countdown before shutdown.
- APCUPSD connection-loss handling.
- Configuration validation.
- Configuration stored in the user's Windows AppData directory.
- Installer and uninstaller for Windows.
- Application version shown in **About**.

## Requirements

- Windows 10/11 x64.
- APCUPSD with the Network Information Server enabled.
- Network access from the PC running UPS Monitor to the APCUPSD NIS port (default `3551`).

## Configuration

Configuration is stored in:

```text
%AppData%\UPS Monitor\config.json
```

Example:

```json
{
  "server": "10.1.100.12:3551",
  "poll_interval": "5s",
  "shutdown": {
    "enabled": true,
    "on_battery_delay": "15s",
    "grace_period": "15s"
  }
}
```

Durations use Go duration syntax, for example:

```text
8s
15m
2h
```

Settings can be changed from the application's **Настройки** window.

The application does not install `config.json` into `Program Files`, so application updates do not overwrite user settings.

## Installation

Use the release installer:

```text
UPS-Monitor-1.0.0-Setup.exe
```

The installer installs the application into:

```text
C:\Program Files\UPS Monitor
```

It creates Start Menu and Desktop shortcuts and registers the application for normal Windows uninstall.

User configuration in `%AppData%\UPS Monitor` is intentionally kept when the application is uninstalled.

## Building from source

Prerequisites:

- Go
- Inno Setup 7

Build the Windows GUI executable:

```powershell
go build -tags migrated_fynedo -ldflags="-H=windowsgui" -o .\UPS-Monitor.exe .\cmd\app
```

Build the installer from `installer\ups-monitor.iss` with Inno Setup Compiler.

Or use the release build script:

```powershell
.\build-release.ps1
```

The script builds the GUI executable and then invokes Inno Setup to create the installer.

## Release

The current release is:

```text
1.0.0
```

Create the Git tag after committing the release:

```powershell
git add .
git commit -m "Release v1.0.0"
git tag -a v1.0.0 -m "UPS Monitor v1.0.0"
git push origin main
git push origin v1.0.0
```

## License

No license has been selected for the project yet.
