using System;
using System.Collections.Generic;
using System.ComponentModel;
using Windows.System.UserProfile;

namespace SavesBinder
{
    public class LangManager : INotifyPropertyChanged
    {
        private string _currentLang = "en"; // English by default

        public string CurrentLang
        {
            get => _currentLang;
            set
            {
                // Ignore null or an empty string that the ComboBox may send during initialization.
                if (!string.IsNullOrEmpty(value) && _currentLang != value)
                {
                    _currentLang = value;
                    PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(nameof(CurrentLang)));
                }
            }
        }

        public event PropertyChangedEventHandler? PropertyChanged;

        private readonly Dictionary<string, Dictionary<string, string>> _translations = new();

        // IMPORTANT: The constructor must stay public so XAML can create the instance.
        public LangManager()
        {
            _translations["en"] = new Dictionary<string, string>
            {
                { "orig_deleted", " (original deleted)" },
                { "link_broken", " (link broken)" },
                { "btn_restore", "Restore" },
                { "btn_unbind", "Unbind" },
                { "btn_destroy", "Delete" },
                { "hint_restore", "Restore: deletes link and database record, moves saves back to original path" },
                { "hint_unbind", "Unbind: deletes link and database record, keeps saves in storage" },
                { "hint_destroy", "Delete: deletes link AND database record AND saves from storage" },
                { "confirm_title", "Confirmation" },
                { "check_confirm", "Confirm deletion" },
                { "lbl_db_file", "Database file:" },
                { "lbl_storage", "Storage:" },
                { "lbl_saves_path", "Saves path:" },
                { "btn_bind", "Bind" },
                { "btn_verify", "Verify & Restore Broken Links" },
                { "lbl_search", "Search" },
                { "search_placeholder", "Filter by path..." },
                { "error_title", "Error" },
                { "success_title", "Done" },
                { "dlg_yes", "Yes" },
                { "dlg_cancel", "Cancel" },
                { "err_no_storage", "Choose a storage folder." },
                { "err_no_saves", "Choose a saves folder." },
                { "err_db_path", "Choose a database file." },
                { "err_db_parse", "Cannot read the database: {0}" },
                { "err_bind_failed", "Cannot bind the folder: {0}" },
                { "err_restore", "Cannot restore saves: {0}" },
                { "err_unbind", "Cannot unbind the folder: {0}" },
                { "err_destroy", "Cannot delete saves: {0}" },
                { "err_operation", "Error: {0}" },
                { "success_bind", "Folder \"{0}\" is bound." },
                { "success_restore", "Saves restored." },
                { "success_unbind", "Folder unbound; saves remain in storage." },
                { "success_destroy", "Saves deleted." },
                { "confirm_restore_msg", "Restore saves from \"{0}\" back to the original location?" },
                { "confirm_unbind_msg", "Unbind \"{0}\"? Saves will remain in storage." },
                { "confirm_destroy_msg", "Delete \"{0}\" together with all saves? This cannot be undone." },
                { "progress_bind", "Moving saves..." },
                { "progress_restore", "Restoring saves..." },
                { "progress_verify", "Restoring links..." },
                { "progress_unbind", "Removing link..." },
                { "progress_destroy", "Deleting saves..." },
                { "verify_ok", "All links are healthy." },
                { "verify_found_msg", "Links to restore: {0}. Restore?" },
                { "verify_restore_err", "Cannot restore links:\n{0}" },
                { "verify_conflict_reason", "a real folder is in the way, not a link" },
                { "settings_tooltip", "Settings" },
                { "settings_title", "Settings" },
                { "settings_language", "Language" },
                { "settings_position", "Window position on startup" },
                { "settings_position_center", "Screen center" },
                { "settings_position_previous", "Last position" },
                { "settings_size", "Window size" },
                { "settings_width", "Width" },
                { "settings_height", "Height" },
                { "err_file_missing", "The specified file does not exist. Check the path." },
                { "err_folder_missing", "The specified folder does not exist. Check the path." },
                { "debug_load_settings", "Failed to load settings: {0}" },
                { "debug_save_settings", "Failed to save settings: {0}" },
                // Add the remaining keys here as needed.
            };

            _translations["ru"] = new Dictionary<string, string>
            {
                { "orig_deleted", " (оригинал удален)" },
                { "link_broken", " (ссылка сломана)" },
                { "btn_restore", "Восстановить" },
                { "btn_unbind", "Отвязать" },
                { "btn_destroy", "Удалить" },
                { "hint_restore", "Восстановление: удаляет ссылку и запись в базе, перемещает сейвы обратно на оригинальное место" },
                { "hint_unbind", "Отвязка: удаляет ссылку и запись в базе, но оставляет папку с сейвами в хранилище" },
                { "hint_destroy", "Удаление: удаляет И ссылку, И запись в базе, И папку с сохранениями из хранилища" },
                { "confirm_title", "Подтверждение" },
                { "check_confirm", "Подтверждать удаление" },
                { "lbl_db_file", "Файл базы данных:" },
                { "lbl_storage", "Хранилище:" },
                { "lbl_saves_path", "Путь к сейвам:" },
                { "btn_bind", "Привязать" },
                { "btn_verify", "Проверить и восстановить неработающие ссылки" },
                { "lbl_search", "Поиск" },
                { "search_placeholder", "Фильтр по пути..." },
                { "error_title", "Ошибка" },
                { "success_title", "Готово" },
                { "dlg_yes", "Да" },
                { "dlg_cancel", "Отмена" },
                { "err_no_storage", "Укажите папку хранилища." },
                { "err_no_saves", "Укажите папку сохранений." },
                { "err_db_path", "Укажите файл базы данных." },
                { "err_db_parse", "Не удалось прочитать базу данных: {0}" },
                { "err_bind_failed", "Не удалось привязать папку: {0}" },
                { "err_restore", "Не удалось восстановить сохранения: {0}" },
                { "err_unbind", "Не удалось отвязать папку: {0}" },
                { "err_destroy", "Не удалось удалить сохранения: {0}" },
                { "err_operation", "Ошибка: {0}" },
                { "success_bind", "Папка \"{0}\" привязана." },
                { "success_restore", "Сохранения восстановлены." },
                { "success_unbind", "Папка отвязана, сохранения остались в хранилище." },
                { "success_destroy", "Сохранения удалены." },
                { "confirm_restore_msg", "Восстановить сохранения из \"{0}\" обратно на исходное место?" },
                { "confirm_unbind_msg", "Отвязать \"{0}\"? Сохранения останутся в хранилище." },
                { "confirm_destroy_msg", "Удалить \"{0}\" вместе со всеми сохранениями? Это действие необратимо." },
                { "progress_bind", "Перемещение сохранений..." },
                { "progress_restore", "Восстановление сохранений..." },
                { "progress_verify", "Восстановление ссылок..." },
                { "progress_unbind", "Удаление ссылки..." },
                { "progress_destroy", "Удаление сохранений..." },
                { "verify_ok", "Все ссылки в порядке." },
                { "verify_found_msg", "Найдено ссылок для восстановления: {0}. Восстановить?" },
                { "verify_restore_err", "Не удалось восстановить ссылки:\n{0}" },
                { "verify_conflict_reason", "на месте ссылки находится обычная папка" },
                { "settings_tooltip", "Настройки" },
                { "settings_title", "Настройки" },
                { "settings_language", "Язык" },
                { "settings_position", "Положение окна при запуске" },
                { "settings_position_center", "По центру экрана" },
                { "settings_position_previous", "Последнее положение" },
                { "settings_size", "Размеры окна" },
                { "settings_width", "Ширина" },
                { "settings_height", "Высота" },
                { "err_file_missing", "Указанный файл не существует. Проверьте путь." },
                { "err_folder_missing", "Указанная папка не существует. Проверьте путь." },
                { "debug_load_settings", "Ошибка загрузки настроек: {0}" },
                { "debug_save_settings", "Ошибка сохранения настроек: {0}" },
                // Add the remaining keys here as needed.
            };

            // Detect the system language on startup.
            CurrentLang = GetDefaultSystemLang();
        }

        public string Get(string key)
        {
            // Guard against null in case the converter passes an empty value.
            if (string.IsNullOrEmpty(key)) return string.Empty;

            // If the language is null for some reason, fall back to English.
            var langCode = _currentLang ?? "en";

            if (_translations.TryGetValue(langCode, out var locale) && locale.TryGetValue(key, out var text))
            {
                return text;
            }
            return _translations["en"].TryGetValue(key, out var fallback) ? fallback : key;
        }

        private string GetDefaultSystemLang()
        {
            try
            {
                var langCode = GlobalizationPreferences.Languages[0];
                return langCode.StartsWith("ru", StringComparison.OrdinalIgnoreCase) ? "ru" : "en";
            }
            catch
            {
                return "en";
            }
        }
    }
}