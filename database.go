package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2/dialog"
)

// LinkPair is one bound pair: original path (Link) → storage path (Target).
// Extra JSON fields (e.g. legacy "type") are ignored on load.
type LinkPair struct {
	Target string `json:"target"`
	Link   string `json:"link"`
}

// AppDatabase represents the custom JSON database structure
type AppDatabase struct {
	Links []LinkPair `json:"links"`
}

// getDbPath retrieves the configured DB path or falls back to the current directory
func (a *SavesBinderApp) getDbPath() string {
	prefs := a.app.Preferences()

	savedPath := prefs.StringWithFallback("db_file_path", "")
	if savedPath != "" {
		return savedPath
	}

	// Default fallback: vals_saves_binder_db.json in the current working directory
	ex, err := os.Executable()
	if err != nil {
		return "vals_saves_binder_db.json"
	}

	return filepath.Join(filepath.Dir(ex), "vals_saves_binder_db.json")
}

// loadData loads the link pairs from the custom JSON file
func (a *SavesBinderApp) loadData() {
	dbPath := a.getDbPath()

	file, err := os.ReadFile(dbPath)
	if err != nil {
		a.db = AppDatabase{Links: []LinkPair{}}
		return
	}

	// If the JSON is corrupted, display an error message without crashing the application
	if err := json.Unmarshal(file, &a.db); err != nil {
		dialog.ShowError(fmt.Errorf(T("err_db_parse"), err), a.window)
		a.db = AppDatabase{Links: []LinkPair{}}
	}
}

// saveData saves the link pairs to the custom JSON file
func (a *SavesBinderApp) saveData() {
	dbPath := a.getDbPath()

	// Check for directory creation error and display a window
	if err := os.MkdirAll(filepath.Dir(dbPath), os.ModePerm); err != nil {
		dialog.ShowError(fmt.Errorf(T("err_db_mkdir"), err), a.window)
		return
	}

	jsonData, err := json.MarshalIndent(a.db, "", "    ")
	if err != nil {
		dialog.ShowError(fmt.Errorf(T("err_db_encode"), err), a.window)
		return
	}

	// Write to a temporary file first, then atomically rename it to the target path.
	// This prevents database corruption if the write is interrupted mid-operation.
	tmpPath := dbPath + ".tmp"
	if err := os.WriteFile(tmpPath, jsonData, 0644); err != nil {
		dialog.ShowError(fmt.Errorf(T("err_db_write"), err), a.window)
		return
	}

	// Atomic rename: either the old file remains intact, or the new file fully replaces it
	if err := os.Rename(tmpPath, dbPath); err != nil {
		// Clean up the temporary file on rename failure
		_ = os.Remove(tmpPath)
		dialog.ShowError(fmt.Errorf(T("err_db_write"), err), a.window)
	}
}
