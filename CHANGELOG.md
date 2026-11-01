# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2026-11-01

### Changed

- **Complete rewrite** – Migrated from Go + Fyne to C# + WinUI 3 / Windows App SDK.
- **Modern UI** – Redesigned interface with Mica backdrop, new icons, and improved layout.
- **Portable by design** – Distributed as a portable folder. Settings are now stored in `settings.json` next to `SavesBinder.exe` with no registry writes.
- **Safer operations** – Atomic JSON database writes, rollback on failed binds, disk space checks, path nesting validation, and stricter broken link detection.
- **Refreshed buttons** – Clearer action layout with improved destructive action styling.

### Added

- **Settings panel** – Change UI language (English/Russian), choose window position on startup (center or last used), and adjust window size from a flyout menu.
- **App icon** – Added application icon for the window and taskbar.
- **Window state persistence** – Remembers window position, size, and maximized state between sessions.
- **Confirmation toggle** – Option to skip confirmation dialogs for faster workflows.
- **Improved tooltips** – Added descriptive tooltips for all actions.

### Notes

- **Breaking change** – This release replaces the previous Go/Fyne version. The database format remains compatible, but the application is built on a completely different technology stack.

## [1.1.0] - 2025-05-29

### Added

- Added a reactive "Search" input that filters the list of bound save pairs in real time as you type.
- The search performs a case-insensitive full-text match across both the original path and the storage path of each pair.
