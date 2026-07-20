package main

import (
	"os"
	"strings"
)

// LangManager handles application translations
type LangManager struct {
	currentLang  string
	translations map[string]map[string]string
}

var lang *LangManager

func initLang() {
	lang = &LangManager{
		currentLang:  "en", // Default fallback
		translations: make(map[string]map[string]string),
	}

	// English translations
	lang.translations["en"] = map[string]string{
		"orig_deleted":        " (original deleted)",
		"link_broken":         " (link broken)",
		"btn_restore":         "Restore",
		"btn_unbind":          "Unbind",
		"btn_destroy":         "Delete",
		"hint_restore":        "Restore: deletes link and database record, moves saves back to original path",
		"hint_unbind":         "Unbind: deletes link and database record, keeps saves in storage",
		"hint_destroy":        "Delete: deletes link AND database record AND saves from storage",
		"confirm_title":       "Confirmation",
		"confirm_restore_msg": "Restore saves to original folder and remove from storage?\n%s",
		"confirm_unbind_msg":  "Unbind only? (Saves will remain in storage)\n%s",
		"confirm_destroy_msg": "DANGER! Delete saves from BOTH original and storage?\n%s",
		"err_delete":          "Failed to remove link: %w",
		"err_restore":         "Restore error: %w",
		"err_destroy":         "Destroy error: %w",
		"check_confirm":       "Confirm deletion",
		"lbl_db_file":         "Database file:",
		"lbl_storage":         "Storage:",
		"lbl_saves_path":      "Saves path:",
		"btn_bind":            "Bind",
		"btn_verify":          "Verify & Restore Broken Links",
		"err_no_storage":      "Specify storage first",
		"err_no_saves":        "Specify full path to saves",
		"err_path_process":    "Failed to process path:\n%w",
		"err_not_folder":      "Path is not a folder or does not exist",
		"err_game_name":       "Failed to determine game name",
		"err_db_parse":        "Failed to parse database file:\n%w",
		"err_db_mkdir":        "Failed to create database directory:\n%w",
		"err_db_encode":       "Failed to encode database content:\n%w",
		"err_db_write":        "Failed to write database file:\n%w",
		"err_restore_mkdir":   "Failed to recreate original folder:\n%w",
		"err_clean_storage":   "Files restored, but failed to clean up storage folder:\n%w",
		"err_move_cleanup":    "failed to remove source file after copying: %w",
		"err_already_exists":  "Folder %s already exists in storage",
		"err_create_storage":  "Failed to create folder in storage:\n%w",
		"err_empty_folder":    "Folder is not empty after transfer (hidden files?)",
		"err_junction":        "Failed to create junction:\nWindows reparse point API returned an error",
		"err_bind_failed":     "Failed to complete binding:\n%w",
		"err_already_reparse": "Path is already a junction or symlink:\n%s",
		"err_nested_paths":    "Storage and saves path must not be nested inside each other",
		"err_already_bound":   "This saves path is already bound in the database",
		"err_name_collision":  "Another bound pair already uses storage folder:\n%s",
		"success_title":       "Success",
		"success_bind":        "Game saves \"%s\" are bound to storage",
		"progress_title":      "Please wait",
		"progress_bind":       "Binding saves…",
		"progress_restore":    "Restoring saves…",
		"progress_destroy":    "Deleting saves…",
		"progress_verify":     "Restoring broken links…",
		"verify_title":        "Verification",
		"verify_ok":           "All links are fine",
		"verify_found_msg":    "Found %d broken links. Restore all?",
		"verify_restore_err":  "Failed to restore some links:\n%s",
		"verify_restore_ok":   "All broken links have been restored",
		"lbl_search":          "Search",
		"search_placeholder":  "Filter by path...",
	}

	// Russian translations
	lang.translations["ru"] = map[string]string{
		"orig_deleted":        " (оригинал удален)",
		"link_broken":         " (ссылка сломана)",
		"btn_restore":         "Восстановить",
		"btn_unbind":          "Отвязать",
		"btn_destroy":         "Удалить",
		"hint_restore":        "Восстановление: удаляет ссылку и запись в базе, перемещает сейвы обратно на оригинальное место",
		"hint_unbind":         "Отвязка: удаляет ссылку и запись в базе, но оставляет папку с сейвами в хранилище",
		"hint_destroy":        "Удаление: удаляет И ссылку, И запись в базе, И папку с сохранениями из хранилища",
		"confirm_title":       "Подтверждение",
		"confirm_restore_msg": "Восстановить сохранения в оригинальную папку и убрать из хранилища?\n%s",
		"confirm_unbind_msg":  "Только отвязать? (Сохранения останутся в хранилище)\n%s",
		"confirm_destroy_msg": "ВНИМАНИЕ! Удалить сохранения ИЗ ХРАНИЛИЩА навсегда?\n%s",
		"err_delete":          "Не удалось удалить ссылку: %w",
		"err_restore":         "Ошибка восстановления: %w",
		"err_destroy":         "Ошибка удаления: %w",
		"check_confirm":       "Подтверждать удаление",
		"lbl_db_file":         "Файл базы данных:",
		"lbl_storage":         "Хранилище:",
		"lbl_saves_path":      "Путь к сейвам:",
		"btn_bind":            "Привязать",
		"btn_verify":          "Проверить и восстановить неработающие ссылки",
		"err_no_storage":      "Сначала укажи хранилище",
		"err_no_saves":        "Укажи полный путь к сохранениям",
		"err_path_process":    "Не удалось обработать путь:\n%w",
		"err_not_folder":      "Путь не является папкой или не существует",
		"err_game_name":       "Не удалось определить имя игры",
		"err_db_parse":        "Не удалось прочитать файл базы данных:\n%w",
		"err_db_mkdir":        "Не удалось создать папку базы данных:\n%w",
		"err_db_encode":       "Не удалось закодировать содержимое базы данных:\n%w",
		"err_db_write":        "Не удалось записать файл базы данных:\n%w",
		"err_restore_mkdir":   "Не удалось пересоздать оригинальную папку:\n%w",
		"err_clean_storage":   "Файлы восстановлены, но не удалось очистить папку в хранилище:\n%w",
		"err_move_cleanup":    "не удалось удалить исходный файл после копирования: %w",
		"err_already_exists":  "Папка %s уже существует в хранилище",
		"err_create_storage":  "Не удалось создать папку в хранилище:\n%w",
		"err_empty_folder":    "После переноса папка не пуста (возможно, скрытые файлы)",
		"err_junction":        "Не удалось создать junction:\nошибка Windows reparse point API",
		"err_bind_failed":     "Не удалось завершить привязку:\n%w",
		"err_already_reparse": "Путь уже является junction или symlink:\n%s",
		"err_nested_paths":    "Хранилище и путь к сейвам не должны быть вложены друг в друга",
		"err_already_bound":   "Этот путь к сейвам уже привязан в базе",
		"err_name_collision":  "Другая привязка уже использует папку в хранилище:\n%s",
		"success_title":       "Успех",
		"success_bind":        "Сохранения игры \"%s\" привязаны к хранилищу",
		"progress_title":      "Подождите",
		"progress_bind":       "Привязка сохранений…",
		"progress_restore":    "Восстановление сохранений…",
		"progress_destroy":    "Удаление сохранений…",
		"progress_verify":     "Восстановление ссылок…",
		"verify_title":        "Проверка",
		"verify_ok":           "Все ссылки в порядке",
		"verify_found_msg":    "Найдено %d сломанных ссылок. Восстановить все?",
		"verify_restore_err":  "Не удалось восстановить некоторые ссылки:\n%s",
		"verify_restore_ok":   "Все сломанные ссылки восстановлены",
		"lbl_search":          "Поиск",
		"search_placeholder":  "Фильтр по пути...",
	}
}

// GetDefaultSystemLang returns ru or en based on OS locale (Windows-aware).
func GetDefaultSystemLang() string {
	// Prefer real OS locale (works on Windows where LANG/LC_ALL are often empty)
	if code, err := systemLanguageCode(); err == nil && code != "" {
		if strings.HasPrefix(strings.ToLower(code), "ru") {
			return "ru"
		}
		return "en"
	}

	// Fallback for unusual environments
	sysLang := os.Getenv("LANG")
	if sysLang == "" {
		sysLang = os.Getenv("LC_ALL")
	}
	if strings.Contains(strings.ToLower(sysLang), "ru") {
		return "ru"
	}
	return "en"
}

// T retrieves the translated string by key
func T(key string) string {
	if lang == nil {
		initLang()
	}
	if locales, ok := lang.translations[lang.currentLang]; ok {
		if text, found := locales[key]; found {
			return text
		}
	}

	// Fallback to English if key not found in current locale
	return lang.translations["en"][key]
}

// SetLanguage allows switching language at runtime
func SetLanguage(locale string) {
	if lang == nil {
		initLang()
	}
	if _, ok := lang.translations[locale]; ok {
		lang.currentLang = locale
	}
}
