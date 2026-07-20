# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - 2026-07-20

### Performance

- Virtualized bound-pairs list (`widget.List`) for smooth scrolling with 50–100+ entries.
- Truncate long paths with ellipsis; full path available via hover tooltip.
- Debounced search filtering; path health (`os.Stat`) cached and scanned off the UI thread.
- Bind / restore / unbind / destroy / verify run filesystem work in background goroutines.

### Stability

- Hover tooltips create popups only on the Fyne UI thread.
- Bind guards: reject already-reparse paths, nested storage/saves paths, and DB collisions.
- Create junctions via Win32 reparse API instead of `cmd /c mklink /J`.
- Progress dialogs for long operations; localized success dialog title; better Windows language detection.

### Dependencies

- Upgrade Fyne `v2.7.4` → `v2.8.0` (and related transitive UI deps).
- Bump `golang.org/x/sys` and other `golang.org/x/*` modules used by the app.

## [1.1.0] - 2025-05-29

### Added

- Added a reactive "Search" input that filters the list of bound save pairs in real time as you type.
- The search performs a case-insensitive full-text match across both the original path and the storage path of each pair.
