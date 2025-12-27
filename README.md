# RayFlow PAK Explorer

A cross-platform GUI tool for browsing and extracting RFPK archive files used by the RayFlow game engine.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)

## Installation

### Download

Download the latest release for your platform from the [Releases](https://github.com/pulsestudio/rfpak-explorer/releases) page:

| Platform | Download |
|----------|----------|
| Windows | `rfpak-explorer-windows-amd64.exe` |
| macOS (Intel) | `rfpak-explorer-darwin-amd64` |
| macOS (Apple Silicon) | `rfpak-explorer-darwin-arm64` |
| Linux | `rfpak-explorer-linux-amd64` |

### macOS / Linux

```bash
# Make executable
chmod +x rfpak-explorer-*

# Run
./rfpak-explorer-darwin-arm64
```

### Windows

Double-click `rfpak-explorer-windows-amd64.exe` to run.

## Usage

1. Click **Open Archive** or use `Ctrl+O`
2. Select a `.pak` file
3. Browse files in the tree view
4. Click on a file to preview
5. Use **Extract Selected** or **Extract All** to save files

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Ctrl+O` | Open archive |
| `Ctrl+Q` | Quit |

## Building from Source

Requires Go 1.21+ and platform-specific dependencies for [Fyne](https://developer.fyne.io/started/).

```bash
git clone https://github.com/pulsestudio/rfpak-explorer.git
cd rfpak-explorer
make build
```

## License

MIT License - see [LICENSE](LICENSE) for details.

