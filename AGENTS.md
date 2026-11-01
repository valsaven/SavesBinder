# SavesBinder

WinUI 3 / Windows App SDK desktop app (C#, XAML, .NET 8), single project. No tests, CI, or lint/format config — a successful build is the only check.

## Build and run

- `dotnet build SavesBinder.csproj -p:Platform=x64` (`x86` or `ARM64` also valid).
- Plain `dotnet build` fails: packaged .NET apps cannot be AnyCPU (`NETSDK1198` / processor-architecture error).
- Debug/run from `SavesBinder.slnx` in Visual Studio with the "SavesBinder (Unpackaged)" profile; settings live next to the exe, so a packaged (MSIX) install cannot save them.
- Publish: `dotnet publish SavesBinder.csproj -c Release -p:Platform=x64` (x86/ARM64 via `-p:Platform=...`) produces a portable folder in `bin\Release\<TFM>\<RID>\publish\` (46 files, ~37 MB). It is framework-dependent: the target machine needs the .NET 8 Desktop Runtime and Windows App Runtime 2.5 installed. No single-file bundle, no `%TEMP%` extraction, no registry writes.

## Architecture notes

- `App.xaml` registers two app-wide resources: `Loc` (`LangManager`) and `T` (`TranslateConverter`). Every localized string in `MainWindow.xaml` uses
  `{Binding Source={StaticResource Loc}, Path=CurrentLang, Converter={StaticResource T}, ConverterParameter=<key>}`;
  changing `CurrentLang` re-evaluates all labels, so never set `Text` directly for UI strings.
- `LangManager.cs` is the only translation store (ru/en dictionaries). Add new UI text to **both** dictionaries using the same key. Missing keys fall back to English, then to the raw key.
- `LinkDatabase.cs` reads the DB JSON: `{ "links": [ { "target": "...", "link": "..." } ] }` (`target` = stored saves path, `link` = symlink at the original location). Missing/null `links` yields an empty list; unknown fields are ignored; property matching is case-insensitive. `SaveItem.TargetText`/`LinkText` are display-only formatted copies.
- The list loads at startup, on Enter in the DB path field, and after picking a DB file.
- Operations live in `MainWindow.xaml.cs` (`Bind`/`VerifyLinks`/`Restore`/`Unbind`/`Destroy`) and run via `RunBusyAsync` → `Task.Run`. Each action re-validates paths with `FileOperations`; the confirmation `ContentDialog` is skipped when `ConfirmCheckBox` is off.
  - Bind: guard paths/disk space/nesting/DB collisions, backup + copy + integrity check (`FileOperations.MoveDirectorySafe`), create a junction, then append the DB entry. Failed junction creation moves the files back.
  - Verify: recreate links whose storage folder exists but whose junction is missing or dangling. `IsReachableDirectory` is used because `Directory.Exists` stays true for a dangling junction. A real folder occupying the link path is reported as a conflict and left untouched.
  - Restore: remove the link, move storage contents back, delete the DB entry. Unbind: remove the link, keep storage, delete the DB entry. Delete: remove the link, recursively delete storage, delete the DB entry.
- `Junction.cs` creates NTFS directory junctions via `FSCTL_SET_REPARSE_POINT` (no admin rights, unlike symlinks). `FileOperations.RemoveLink` deletes reparse points without following them and refuses to delete a real non-empty directory.
- DB writes are atomic (`<db>.tmp` + `File.Move(..., overwrite: true)` in `LinkDatabase.Save`). Bind refuses a DB file inside the saves or storage target folder.
- `SaveItem` uses `target`/`link`; legacy extra JSON fields are ignored on load.
- Settings are JSON in `settings.json` next to the exe (`AppContext.BaseDirectory`), loaded at startup and saved on window close. Moving the whole publish folder moves the settings with it; a read-only location (Program Files) silently loses them. Besides paths, language and `ConfirmActions` they store window geometry (`WindowX/Y/Width/Height`, `IsMaximized`). Only restored-state bounds are recorded (tracked via `AppWindow.Changed`); on restore, an off-screen position is centered on the primary display.
- File/folder pickers require `WinRT.Interop.InitializeWithWindow.Initialize(picker, hwnd)` (see `PickFolderAsync` and `BrowseDbFile_Click`); picking fails otherwise.
