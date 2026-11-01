using Microsoft.UI.Windowing;
using Microsoft.UI.Xaml;
using Microsoft.UI.Xaml.Controls;
using Microsoft.UI.Xaml.Input;
using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text.Json;
using System.Threading.Tasks;
using Windows.Graphics;
using Windows.Storage.Pickers;
using Windows.System;

namespace SavesBinder
{
    public sealed partial class MainWindow : Window
    {
        private LinkDatabase _database = new();
        private List<SaveItem> _allSaves = new();

        // Window bounds in the normal (non-maximized) state, restored after closing.
        private RectInt32 _normalBounds;
        private bool _isMaximized;

        // Settings live next to the exe so the whole app folder can be moved.
        private readonly string _settingsFilePath;

        private LangManager Loc => (LangManager)Application.Current.Resources["Loc"];

        public MainWindow()
        {
            this.InitializeComponent();

            var version = System.Reflection.Assembly.GetExecutingAssembly().GetName().Version;
            Title = version == null ? "Val's Saves Binder" : $"Val's Saves Binder v{version.Major}.{version.Minor}.{version.Build}";

            // Window and taskbar icon (the exe icon is set in the csproj).
            AppWindow.SetIcon(Path.Combine(AppContext.BaseDirectory, "Assets", "floppy-icon.ico"));

            // Determine the settings file path.
            _settingsFilePath = Path.Combine(AppContext.BaseDirectory, "settings.json");

            // Load settings BEFORE loading the data.
            var settings = LoadSettings();
            RestoreWindowState(settings);

            LoadLinks();

            AppWindow.Changed += AppWindow_Changed;

            // Save settings when the window closes.
            this.Closed += MainWindow_Closed;
        }

        // --- SETTINGS LOGIC ---

        private AppSettings? LoadSettings()
        {
            try
            {
                if (File.Exists(_settingsFilePath))
                {
                    string json = File.ReadAllText(_settingsFilePath);
                    var settings = JsonSerializer.Deserialize<AppSettings>(json);

                    if (settings != null)
                    {
                        // Restore the paths.
                        DbFileTextBox.Text = settings.DbFilePath;
                        StorageTextBox.Text = settings.StoragePath;
                        SavesPathTextBox.Text = settings.SavesPath;
                        ConfirmCheckBox.IsChecked = settings.ConfirmActions;
                        WindowPositionComboBox.SelectedIndex = settings.CenterOnStartup ? 0 : 1;

                        // Restore the language.
                        if (Application.Current.Resources["Loc"] is LangManager locManager)
                        {
                            locManager.CurrentLang = settings.Language;
                        }

                        return settings;
                    }
                }
            }
            catch (Exception ex)
            {
                // If the file is corrupted or an error occurs, just ignore it (defaults will be used).
                System.Diagnostics.Debug.WriteLine(string.Format(Loc.Get("debug_load_settings"), ex.Message));
            }

            return null;
        }

        private void RestoreWindowState(AppSettings? settings)
        {
            if (settings is { WindowWidth: > 0, WindowHeight: > 0 })
            {
                var bounds = new RectInt32(settings.WindowX, settings.WindowY, settings.WindowWidth, settings.WindowHeight);

                // Centered always; otherwise if the window no longer lands on any monitor, center it too.
                var targetDisplay = settings.CenterOnStartup
                    ? null
                    : DisplayArea.GetFromPoint(new PointInt32(bounds.X, bounds.Y), DisplayAreaFallback.None);
                var display = targetDisplay ?? DisplayArea.Primary;
                if (targetDisplay is null)
                {
                    bounds.X = display.WorkArea.X + Math.Max(0, (display.WorkArea.Width - bounds.Width) / 2);
                    bounds.Y = display.WorkArea.Y + Math.Max(0, (display.WorkArea.Height - bounds.Height) / 2);
                }

                bounds.Width = Math.Min(bounds.Width, display.WorkArea.Width);
                bounds.Height = Math.Min(bounds.Height, display.WorkArea.Height);

                // ponytail: bounds are physical pixels; a different DPI/scale on next launch may shift the window slightly.
                AppWindow.MoveAndResize(bounds);
            }

            _normalBounds = new RectInt32(AppWindow.Position.X, AppWindow.Position.Y, AppWindow.Size.Width, AppWindow.Size.Height);
            _isMaximized = settings?.IsMaximized == true;

            if (_isMaximized && AppWindow.Presenter is OverlappedPresenter presenter)
            {
                presenter.Maximize();
            }
        }

        private void AppWindow_Changed(AppWindow sender, AppWindowChangedEventArgs args)
        {
            if (AppWindow.Presenter is not OverlappedPresenter presenter)
            {
                return;
            }

            // Remember the "maximized" state even when closing from a minimized window.
            if (presenter.State != OverlappedPresenterState.Minimized)
            {
                _isMaximized = presenter.State == OverlappedPresenterState.Maximized;
            }

            if (presenter.State == OverlappedPresenterState.Restored && (args.DidPositionChange || args.DidSizeChange))
            {
                _normalBounds = new RectInt32(sender.Position.X, sender.Position.Y, sender.Size.Width, sender.Size.Height);
            }
        }

        private void SaveSettings()
        {
            try
            {
                var settings = new AppSettings
                {
                    DbFilePath = DbFileTextBox.Text,
                    StoragePath = StorageTextBox.Text,
                    SavesPath = SavesPathTextBox.Text,
                    Language = Loc.CurrentLang,
                    WindowX = _normalBounds.X,
                    WindowY = _normalBounds.Y,
                    WindowWidth = _normalBounds.Width,
                    WindowHeight = _normalBounds.Height,
                    IsMaximized = _isMaximized,
                    ConfirmActions = ConfirmCheckBox.IsChecked == true,
                    CenterOnStartup = WindowPositionComboBox.SelectedIndex == 0
                };

                string json = JsonSerializer.Serialize(settings, new JsonSerializerOptions { WriteIndented = true });
                File.WriteAllText(_settingsFilePath, json);
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine(string.Format(Loc.Get("debug_save_settings"), ex.Message));
            }
        }

        private void MainWindow_Closed(object sender, WindowEventArgs args)
        {
            SaveSettings();
        }

        // --- WINDOW: SIZE FROM SETTINGS ---

        private bool _updatingWindowSize;

        private void SettingsFlyout_Opened(object sender, object e)
        {
            var workArea = DisplayArea.GetFromWindowId(AppWindow.Id, DisplayAreaFallback.Primary).WorkArea;

            _updatingWindowSize = true;
            WindowWidthBox.Maximum = workArea.Width;
            WindowHeightBox.Maximum = workArea.Height;
            WindowWidthBox.Value = _normalBounds.Width;
            WindowHeightBox.Value = _normalBounds.Height;
            _updatingWindowSize = false;
        }

        private void WindowSizeBox_ValueChanged(NumberBox sender, NumberBoxValueChangedEventArgs args)
        {
            if (_updatingWindowSize)
            {
                return;
            }

            if (double.IsNaN(WindowWidthBox.Value) || double.IsNaN(WindowHeightBox.Value))
            {
                return;
            }

            var display = DisplayArea.GetFromWindowId(AppWindow.Id, DisplayAreaFallback.Primary);
            var workArea = display.WorkArea;
            int width = (int)WindowWidthBox.Value;
            int height = (int)WindowHeightBox.Value;

            // NumberBox already shows an error for out-of-range values; here we simply do not apply them.
            if (width <= 0 || height <= 0 || width > workArea.Width || height > workArea.Height)
            {
                return;
            }

            if (AppWindow.Presenter is OverlappedPresenter { State: OverlappedPresenterState.Maximized } presenter)
            {
                presenter.Restore();
            }

            // Always center the window in the work area so it cannot be moved off screen.
            AppWindow.MoveAndResize(new RectInt32(
                workArea.X + Math.Max(0, (workArea.Width - width) / 2),
                workArea.Y + Math.Max(0, (workArea.Height - height) / 2),
                width,
                height));
            _normalBounds = AppWindow.Position == default && AppWindow.Size == default
                ? new RectInt32(workArea.X + (workArea.Width - width) / 2, workArea.Y + (workArea.Height - height) / 2, width, height)
                : new RectInt32(AppWindow.Position.X, AppWindow.Position.Y, AppWindow.Size.Width, AppWindow.Size.Height);
        }

        // --- LOAD THE LIST FROM THE DATABASE ---

        private void LoadLinks()
        {
            _database = new LinkDatabase();
            _allSaves = new List<SaveItem>();

            var path = DbFileTextBox.Text;
            if (!string.IsNullOrWhiteSpace(path))
            {
                try
                {
                    _database = LinkDatabase.Load(path);
                    _allSaves = _database.Links.ToList(); // operations mutate the DB list on a worker thread
                    HideError();
                }
                catch (Exception ex) when (ex is IOException or UnauthorizedAccessException or JsonException or ArgumentException)
                {
                    ShowError(string.Format(Loc.Get("err_db_parse"), ex.Message));
                }
            }

            ApplyFilter();
        }

        private void RemoveFromDatabase(SaveItem item, string dbPath)
        {
            _database.Links.RemoveAll(link =>
                string.Equals(link.Link, item.Link, StringComparison.OrdinalIgnoreCase) &&
                string.Equals(link.Target, item.Target, StringComparison.OrdinalIgnoreCase));
            _database.Save(dbPath);
        }

        // --- REACTIVE SEARCH LOGIC ---

        private void SearchTextBox_TextChanged(object sender, TextChangedEventArgs e)
        {
            ApplyFilter();
        }

        private void ApplyFilter()
        {
            var searchText = SearchTextBox.Text;

            // Filter the WHOLE master list, ignoring case (OrdinalIgnoreCase)
            SavesListView.ItemsSource = string.IsNullOrWhiteSpace(searchText)
                ? _allSaves
                : _allSaves.Where(save =>
                    save.LinkText.Contains(searchText, StringComparison.OrdinalIgnoreCase) ||
                    save.TargetText.Contains(searchText, StringComparison.OrdinalIgnoreCase)
                ).ToList();
        }

        // --- LIST BUTTONS ---

        private async void RestoreButton_Click(object sender, RoutedEventArgs e)
        {
            if (sender is not Button { DataContext: SaveItem item })
            {
                return;
            }

            var dbPath = DbFileTextBox.Text.Trim();
            if (dbPath.Length == 0)
            {
                ShowError(Loc.Get("err_db_path"));
                return;
            }

            if (!await ConfirmAsync("confirm_restore_msg", item.Link))
            {
                return;
            }

            try
            {
                await RunBusyAsync("progress_restore", () => Restore(item, dbPath));
                LoadLinks();
                ShowInfoBar(Loc.Get("success_restore"), InfoBarSeverity.Success);
            }
            catch (Exception ex)
            {
                ShowError(string.Format(Loc.Get("err_restore"), ex.Message));
            }
        }

        private async void UnbindButton_Click(object sender, RoutedEventArgs e)
        {
            if (sender is not Button { DataContext: SaveItem item })
            {
                return;
            }

            var dbPath = DbFileTextBox.Text.Trim();
            if (dbPath.Length == 0)
            {
                ShowError(Loc.Get("err_db_path"));
                return;
            }

            if (!await ConfirmAsync("confirm_unbind_msg", item.Link))
            {
                return;
            }

            try
            {
                await RunBusyAsync("progress_unbind", () => Unbind(item, dbPath));
                LoadLinks();
                ShowInfoBar(Loc.Get("success_unbind"), InfoBarSeverity.Success);
            }
            catch (Exception ex)
            {
                ShowError(string.Format(Loc.Get("err_unbind"), ex.Message));
            }
        }

        private async void DeleteButton_Click(object sender, RoutedEventArgs e)
        {
            if (sender is not Button { DataContext: SaveItem item })
            {
                return;
            }

            var dbPath = DbFileTextBox.Text.Trim();
            if (dbPath.Length == 0)
            {
                ShowError(Loc.Get("err_db_path"));
                return;
            }

            if (!await ConfirmAsync("confirm_destroy_msg", item.Link))
            {
                return;
            }

            try
            {
                await RunBusyAsync("progress_destroy", () => Destroy(item, dbPath));
                LoadLinks();
                ShowInfoBar(Loc.Get("success_destroy"), InfoBarSeverity.Success);
            }
            catch (Exception ex)
            {
                ShowError(string.Format(Loc.Get("err_destroy"), ex.Message));
            }
        }

        private async void BindButton_Click(object sender, RoutedEventArgs e)
        {
            var storage = StorageTextBox.Text.Trim();
            var saves = SavesPathTextBox.Text.Trim();
            var dbPath = DbFileTextBox.Text.Trim();

            if (storage.Length == 0)
            {
                ShowError(Loc.Get("err_no_storage"));
                return;
            }

            if (saves.Length == 0)
            {
                ShowError(Loc.Get("err_no_saves"));
                return;
            }

            if (dbPath.Length == 0)
            {
                ShowError(Loc.Get("err_db_path"));
                return;
            }

            try
            {
                await RunBusyAsync("progress_bind", () => Bind(storage, saves, dbPath));
                SavesPathTextBox.Text = string.Empty;
                LoadLinks();

                var gameName = Path.GetFileName(saves.TrimEnd(Path.DirectorySeparatorChar, Path.AltDirectorySeparatorChar));
                ShowInfoBar(string.Format(Loc.Get("success_bind"), gameName), InfoBarSeverity.Success);
            }
            catch (Exception ex)
            {
                ShowError(string.Format(Loc.Get("err_bind_failed"), ex.Message));
            }
        }

        private async void VerifyButton_Click(object sender, RoutedEventArgs e)
        {
            var restorable = new List<SaveItem>();
            var conflicts = new List<string>();
            var conflictReason = Loc.Get("verify_conflict_reason");

            foreach (var save in _allSaves.Where(save => Directory.Exists(save.Target)))
            {
                var attributes = FileOperations.GetAttributesOrNull(save.Link);
                if (attributes is null)
                {
                    restorable.Add(save); // link missing entirely
                }
                else if ((attributes.Value & System.IO.FileAttributes.ReparsePoint) == 0 ||
                         (attributes.Value & System.IO.FileAttributes.Directory) == 0)
                {
                    // A real folder/file sits where the junction belongs: never delete it, just report.
                    conflicts.Add($"{save.Link}: {conflictReason}");
                }
                else if (!FileOperations.IsReachableDirectory(save.Link))
                {
                    restorable.Add(save); // dangling junction
                }
                // ponytail: a junction pointing at a different folder counts as healthy; compare reparse targets if that ever happens.
            }

            if (restorable.Count == 0)
            {
                if (conflicts.Count > 0)
                {
                    ShowError(string.Format(Loc.Get("verify_restore_err"), string.Join(Environment.NewLine, conflicts)));
                }
                else
                {
                    ShowInfoBar(Loc.Get("verify_ok"), InfoBarSeverity.Success);
                }

                return;
            }

            if (!await ConfirmAsync("verify_found_msg", restorable.Count.ToString()))
            {
                return;
            }

            var failures = new List<string>();
            try
            {
                await RunBusyAsync("progress_verify", () => VerifyLinks(restorable, failures));
            }
            catch (Exception ex)
            {
                ShowError(string.Format(Loc.Get("err_operation"), ex.Message));
                return;
            }

            LoadLinks();

            failures.AddRange(conflicts);
            if (failures.Count > 0)
            {
                ShowError(string.Format(Loc.Get("verify_restore_err"), string.Join(Environment.NewLine, failures)));
            }
            else
            {
                ShowInfoBar(Loc.Get("verify_ok"), InfoBarSeverity.Success);
            }
        }

        // --- FILE OPERATIONS (run on a background thread) ---

        private void Bind(string storageDir, string originalPath, string dbPath)
        {
            if (!Directory.Exists(storageDir))
            {
                throw new DirectoryNotFoundException($"Storage folder not found: {storageDir}");
            }

            var originalAbs = FileOperations.Normalize(originalPath);
            FileOperations.ValidatePath(originalAbs);
            FileOperations.CheckWritePermission(storageDir);

            if (!Directory.Exists(originalAbs))
            {
                throw new DirectoryNotFoundException($"Saves folder not found: {originalAbs}");
            }

            var gameName = Path.GetFileName(originalAbs);
            if (string.IsNullOrWhiteSpace(gameName))
            {
                throw new InvalidOperationException("Cannot determine the game folder name.");
            }

            var newDir = Path.Combine(FileOperations.Normalize(storageDir), gameName);
            var dbFullPath = FileOperations.Normalize(dbPath);
            if (FileOperations.IsNestedPath(originalAbs, dbFullPath) || FileOperations.IsNestedPath(newDir, dbFullPath))
            {
                throw new InvalidOperationException("The database file must not be inside the saves or storage target folder.");
            }

            FileOperations.CheckDiskSpace(originalAbs, newDir);

            if (FileOperations.IsNestedPath(storageDir, originalAbs))
            {
                throw new InvalidOperationException("Storage and saves paths must not be nested.");
            }

            if (FileOperations.IsReparsePoint(originalAbs))
            {
                throw new InvalidOperationException($"Path is already a link: {originalAbs}");
            }

            foreach (var existing in _database.Links)
            {
                if (string.Equals(FileOperations.Normalize(existing.Link), originalAbs, StringComparison.OrdinalIgnoreCase))
                {
                    throw new InvalidOperationException("This folder is already bound.");
                }

                if (string.Equals(FileOperations.Normalize(existing.Target), newDir, StringComparison.OrdinalIgnoreCase))
                {
                    throw new InvalidOperationException($"Name collision in storage: {newDir}");
                }
            }

            if (Directory.Exists(newDir) || File.Exists(newDir))
            {
                throw new IOException($"Destination already exists: {newDir}");
            }

            var backupRoot = Path.Combine(FileOperations.Normalize(storageDir), ".backup");
            var backupPath = FileOperations.MoveDirectorySafe(originalAbs, newDir, backupRoot);

            try
            {
                Junction.Create(originalAbs, newDir);
            }
            catch (Exception ex)
            {
                // Junction was not created - move the files back to their original location.
                try
                {
                    if (Directory.Exists(newDir))
                    {
                        var rollbackBackup = FileOperations.MoveDirectorySafe(newDir, originalAbs, backupRoot);
                        FileOperations.TryDeleteDirectory(rollbackBackup);
                    }

                    FileOperations.TryDeleteDirectory(backupPath);
                }
                catch
                {
                    // Rollback failed - the backup and data are left in storage.
                }

                throw new InvalidOperationException(
                    $"Cannot create a junction at {originalAbs}. Saves remain in {newDir}. {ex.Message}", ex);
            }

            FileOperations.TryDeleteDirectory(backupPath);

            _database.Links.Add(new SaveItem { Target = newDir, Link = originalAbs });
            _database.Save(dbPath);
        }

        private static void VerifyLinks(List<SaveItem> broken, List<string> failures)
        {
            foreach (var item in broken)
            {
                try
                {
                    // A dangling junction still occupies the path, so clear it before recreating.
                    FileOperations.RemoveLink(item.Link);
                    Junction.Create(item.Link, item.Target);
                }
                catch (Exception ex)
                {
                    failures.Add($"{item.Link}: {ex.Message}");
                }
            }
        }

        private void Restore(SaveItem item, string dbPath)
        {
            FileOperations.ValidatePath(item.Link);
            FileOperations.ValidatePath(item.Target);

            if (FileOperations.IsNestedPath(item.Link, item.Target))
            {
                throw new InvalidOperationException("Link and target must not be nested.");
            }

            FileOperations.CheckWritePermission(Path.GetDirectoryName(FileOperations.Normalize(item.Link))!);

            if (Directory.Exists(item.Target))
            {
                if (FileOperations.IsReparsePoint(item.Target))
                {
                    throw new InvalidOperationException($"Storage path is a link; refusing to move it: {item.Target}");
                }

                FileOperations.CheckDiskSpace(item.Target, item.Link);
            }

            FileOperations.RemoveLink(item.Link);

            if (Directory.Exists(item.Target))
            {
                Directory.CreateDirectory(item.Link);

                var backupRoot = Path.Combine(
                    Path.GetDirectoryName(FileOperations.Normalize(item.Target))!, ".backup");
                var backupPath = FileOperations.MoveDirectorySafe(item.Target, item.Link, backupRoot);
                FileOperations.TryDeleteDirectory(backupPath);
            }

            RemoveFromDatabase(item, dbPath);
        }

        private void Unbind(SaveItem item, string dbPath)
        {
            FileOperations.ValidatePath(item.Link);
            FileOperations.ValidatePath(item.Target);

            if (FileOperations.IsNestedPath(item.Link, item.Target))
            {
                throw new InvalidOperationException("Link and target must not be nested.");
            }

            FileOperations.RemoveLink(item.Link);
            RemoveFromDatabase(item, dbPath);
        }

        private void Destroy(SaveItem item, string dbPath)
        {
            FileOperations.ValidatePath(item.Link);
            FileOperations.ValidatePath(item.Target);

            if (FileOperations.IsNestedPath(item.Link, item.Target))
            {
                throw new InvalidOperationException("Link and target must not be nested.");
            }

            if (Directory.Exists(item.Target) && FileOperations.IsReparsePoint(item.Target))
            {
                throw new InvalidOperationException($"Storage path is a link; refusing recursive delete: {item.Target}");
            }

            FileOperations.RemoveLink(item.Link);

            if (Directory.Exists(item.Target))
            {
                Directory.Delete(item.Target, recursive: true);
            }
            else if (File.Exists(item.Target))
            {
                File.Delete(item.Target);
            }

            RemoveFromDatabase(item, dbPath);
        }

        // --- UI HELPER METHODS ---

        private async Task RunBusyAsync(string messageKey, Action action)
        {
            BusyText.Text = Loc.Get(messageKey);
            BusyOverlay.Visibility = Visibility.Visible;

            try
            {
                await Task.Run(action);
            }
            finally
            {
                BusyOverlay.Visibility = Visibility.Collapsed;
            }
        }

        private async Task<bool> ConfirmAsync(string messageKey, string argument)
        {
            if (ConfirmCheckBox.IsChecked != true)
            {
                return true;
            }

            var dialog = new ContentDialog
            {
                Title = Loc.Get("confirm_title"),
                Content = string.Format(Loc.Get(messageKey), argument),
                PrimaryButtonText = Loc.Get("dlg_yes"),
                CloseButtonText = Loc.Get("dlg_cancel"),
                DefaultButton = ContentDialogButton.Close,
                XamlRoot = RootGrid.XamlRoot
            };

            return await dialog.ShowAsync() == ContentDialogResult.Primary;
        }

        private void ShowInfoBar(string message, InfoBarSeverity severity)
        {
            ErrorInfoBar.Title = Loc.Get(severity == InfoBarSeverity.Success ? "success_title" : "error_title");
            ErrorInfoBar.Severity = severity;
            ErrorInfoBar.Message = message;
            ErrorInfoBar.IsOpen = true;
        }

        private void ShowError(string message)
        {
            ShowInfoBar(message, InfoBarSeverity.Error);
        }

        private void HideError()
        {
            ErrorInfoBar.IsOpen = false;
        }

        // --- FILE AND FOLDER PICKER HANDLERS ---

        private async void BrowseDbFile_Click(object sender, RoutedEventArgs e)
        {
            var filePicker = new FileOpenPicker();

            // Bind the dialog to the current window (required for WinUI 3).
            var hwnd = WinRT.Interop.WindowNative.GetWindowHandle(this);
            WinRT.Interop.InitializeWithWindow.Initialize(filePicker, hwnd);

            filePicker.FileTypeFilter.Add(".json");
            filePicker.SuggestedStartLocation = PickerLocationId.DocumentsLibrary;

            var file = await filePicker.PickSingleFileAsync();

            if (file != null)
            {
                DbFileTextBox.Text = file.Path;
                HideError(); // Hide the error after a successful pick.
                LoadLinks();
            }
        }

        private async void BrowseStorageFolder_Click(object sender, RoutedEventArgs e)
        {
            await PickFolderAsync(StorageTextBox);
        }

        private async void BrowseSavesFolder_Click(object sender, RoutedEventArgs e)
        {
            await PickFolderAsync(SavesPathTextBox);
        }

        private async Task PickFolderAsync(TextBox targetTextBox)
        {
            var folderPicker = new FolderPicker();

            var hwnd = WinRT.Interop.WindowNative.GetWindowHandle(this);
            WinRT.Interop.InitializeWithWindow.Initialize(folderPicker, hwnd);

            folderPicker.FileTypeFilter.Add("*"); // Required for FolderPicker

            var folder = await folderPicker.PickSingleFolderAsync();

            if (folder != null)
            {
                targetTextBox.Text = folder.Path;
                HideError(); // Hide the error after a successful pick.
            }
        }

        // --- ENTER KEY HANDLERS ---

        private void DbFileTextBox_KeyDown(object sender, KeyRoutedEventArgs e)
        {
            if (e.Key == VirtualKey.Enter)
            {
                e.Handled = true; // Stop further processing (Windows error beep)
                if (ValidateDbFile())
                {
                    LoadLinks();
                }
            }
        }

        private void FolderTextBox_KeyDown(object sender, KeyRoutedEventArgs e)
        {
            if (e.Key == VirtualKey.Enter)
            {
                e.Handled = true; // Stop further processing (Windows error beep)
                if (sender is TextBox textBox)
                {
                    ValidateFolder(textBox);
                }
            }
        }

        // --- FOCUS LOST (LostFocus) HANDLERS ---

        private void DbFileTextBox_LostFocus(object sender, RoutedEventArgs e)
        {
            // Do not show an error if the field is empty (the user simply clicked elsewhere).
            if (!string.IsNullOrWhiteSpace(DbFileTextBox.Text))
            {
                ValidateDbFile();
            }
            else
            {
                HideError();
            }
        }

        private void FolderTextBox_LostFocus(object sender, RoutedEventArgs e)
        {
            if (sender is TextBox textBox && !string.IsNullOrWhiteSpace(textBox.Text))
            {
                ValidateFolder(textBox);
            }
            else
            {
                HideError();
            }
        }

        // Hide the error as soon as the user starts typing.
        private void InputField_TextChanged(object sender, TextChangedEventArgs e)
        {
            HideError();
        }

        // --- VALIDATION LOGIC ---

        private bool ValidateDbFile()
        {
            var path = DbFileTextBox.Text;

            // Check: the path is not empty and the file exists.
            if (string.IsNullOrWhiteSpace(path) || !File.Exists(path))
            {
                ShowError(Loc.Get("err_file_missing"));
                return false;
            }

            HideError();
            return true;
        }

        private void ValidateFolder(TextBox textBox)
        {
            var path = textBox.Text;

            // Check: the path is not empty and the folder exists.
            if (string.IsNullOrWhiteSpace(path) || !Directory.Exists(path))
            {
                ShowError(Loc.Get("err_folder_missing"));
            }
            else
            {
                HideError();
            }
        }
    }
}
