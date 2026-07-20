# SavesBinder

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Windows-lightgrey.svg)](https://www.microsoft.com/windows)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

A lightweight Windows GUI tool designed to centralize your game saves using NTFS directory junctions.

Move your saves to a single secure directory while keeping the original paths fully functional.

## Created by gamer for gamers :3

*Especially for those who hate losing save folders after reinstalling Windows or swapping drives.*

**Tired of saves being scattered everywhere?** One game stores saves in `Documents`, another in `Documents\My Games`, a third in `AppData` and a fourth in some weird custom path?! 🤯

With **SavesBinder**, you can finally gather all your saves into one clean, central folder. The tool keeps the original paths working via junctions and allows you to restore them with a single click-even after a clean system reinstallation.

---

## Features

- **Effortless Binding** - Select a game's save folder: it automatically moves to your central storage and leaves a working junction behind.
- **Centralized Management** - Keep all your precious saves in one dedicated place for easy backups.
- **Broken Link Detection** - Instantly scan, verify and repair broken junctions with a single click (convenient after reinstalling the system).
- **Native NTFS Junctions** - Creates real directory junctions via the Windows reparse-point API.
- **Clean Fyne GUI** - Simple, intuitive and modern lightweight interface.
- **Human-Readable Config** - Links and paths are stored in a simple, portable JSON file.
- **Safe Operations** - Built-in confirmation dialogs prevent accidental data loss.

---

## How It Works

```text
[Original Path] (e.g., AppData\Game) ──(Moves Files)──> [Central Storage] (e.g., D:\GameSaves\Game)
       │                                                               ▲
       └───────────────────(Creates NTFS Junction)─────────────────────┘

```

1. You set a main **Storage folder** (e.g., `D:\GameSaves`).
2. Select the current folder where a game keeps its saves.
3. The program:
   - Moves the saves to `D:\GameSaves\GAME_NAME`.
   - Creates an **NTFS Junction** at the original path pointing to the new location.
4. The game continues to work seamlessly, unaware of the relocation, while your files are safely centralized.

---

## Usage

1. Specify your **Central Storage Folder**.
2. Provide the full path to the game's current save directory.
3. Click **Bind**.
4. Done! The game will keep loading and saving using its original path.

*Note: Use the **Verify & Restore Broken Links** button to automatically scan your database and re-create junctions after a Windows reinstall.*

---

## Important Notes

- Junctions require an **NTFS** volume (the default for Windows system drives).
- While the file transfer process is fully safe, it is always recommended to make a manual backup the first time you try it on a new game.
- **Do not delete junction folders manually** via File Explorer - always use the **Unbind** or **Restore** buttons inside the application to manage them safely.

---

## Screenshots

![Screenshot](/screenshot.png)

---

## Requirements

- **Windows 10 / 11**
- NTFS-formatted storage drives
- To build from source: **Go 1.26+**, a C compiler for CGO (e.g. MinGW/`gcc`) and Git

---

## Installation & Build

### Pre-built Binary

Download the latest `SavesBinder.exe` from [Releases](https://github.com/valsaven/SavesBinder/releases) and run it. No installation required.

### Building from Source

If you prefer to compile the application yourself, make sure you have Go installed:

1. Clone the repository:

   ```shell
   git clone git@github.com:valsaven/SavesBinder.git
   cd SavesBinder
   ```

2. Install dependencies:

   ```shell
   go install fyne.io/tools/cmd/fyne@latest
   go install github.com/akavel/rsrc@latest

   go mod tidy
   ```

3. Generate icons and Windows resources:

   ```shell
   # Embeds PNG icon for the Fyne app window
   fyne bundle -o bundled.go Icon.png

   # Generates Windows PE resource for the .exe file icon
   rsrc -ico floppy-icon.ico -o rsrc.syso
   ```

4. Compile the production build (hides the background console window and optimizes file size):

   ```shell
   go build -ldflags="-s -w -H=windowsgui" -o SavesBinder.exe .
   ```

   or via Fyne package:

   ```shell
   fyne package -release -os windows
   ```

5. *(Optional)* Further compress the binary using UPX:

   ```shell
   upx --best --lzma ".\Saves Binder.exe"
   ```

---

## License

This project is licensed under the MIT License - feel free to use, modify, and distribute it.
