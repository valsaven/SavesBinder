package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

// verifyLinks checks for broken links and prompts for restoration
func (a *SavesBinderApp) verifyLinks() {
	// Prefer cached health when available; fall back to a quick scan for pairs
	// that have not been checked yet.
	var restorable []LinkPair
	a.statusMu.RLock()
	cache := a.pathHealthCache
	for _, p := range a.db.Links {
		if h, ok := cache[pairCacheKey(p)]; ok && h.Checked {
			if !h.TargetMissing && h.LinkMissing {
				restorable = append(restorable, p)
			}
			continue
		}
		// Uncached: cheap sync check only for verify action
		_, errTarget := os.Stat(p.Target)
		_, errLink := os.Stat(p.Link)
		if errTarget == nil && os.IsNotExist(errLink) {
			restorable = append(restorable, p)
		}
	}
	a.statusMu.RUnlock()

	if len(restorable) == 0 {
		dialog.ShowInformation(T("verify_title"), T("verify_ok"), a.window)
		return
	}

	msg := fmt.Sprintf(T("verify_found_msg"), len(restorable))
	dialog.ShowConfirm(T("btn_verify"), msg, func(yes bool) {
		if !yes {
			return
		}

		prog := dialog.NewProgressInfinite(T("progress_title"), T("progress_verify"), a.window)
		prog.Show()

		go func(pairs []LinkPair) {
			var failed []string
			for _, p := range pairs {
				success, errStr := a.restoreOne(p)
				if !success {
					failed = append(failed, fmt.Sprintf("%s: %s", p.Link, errStr))
				}
			}

			fyne.Do(func() {
				prog.Hide()
				a.invalidatePathHealth()
				a.refreshList()

				if len(failed) > 0 {
					errLines := strings.Join(failed, "\n")
					dialog.ShowError(fmt.Errorf(T("verify_restore_err"), errLines), a.window)
				} else {
					dialog.ShowInformation(T("verify_title"), T("verify_ok"), a.window)
				}
			})
		}(restorable)
	}, a.window)
}

// removeFromDB handles removing the record from JSON and refreshing UI.
// Must be called on the UI thread.
func (a *SavesBinderApp) removeFromDB(p LinkPair) {
	var newLinks []LinkPair
	for _, item := range a.db.Links {
		if item.Link != p.Link || item.Target != p.Target {
			newLinks = append(newLinks, item)
		}
	}
	a.db.Links = newLinks
	a.saveData()
	a.invalidatePathHealth()
	a.refreshList()
}

// restoreSave moves files back to original location and removes the database entry
func (a *SavesBinderApp) restoreSave(p LinkPair) {
	process := func() {
		prog := dialog.NewProgressInfinite(T("progress_title"), T("progress_restore"), a.window)
		prog.Show()

		go func() {
			var uiErr error
			var cleanWarn error

			// 1. Remove junction link safely
			if err := os.Remove(p.Link); err != nil && !os.IsNotExist(err) {
				uiErr = fmt.Errorf(T("err_delete"), err)
			} else if _, err := os.Stat(p.Target); err == nil {
				// 2. Move files back if target exists
				if err := os.MkdirAll(p.Link, os.ModePerm); err != nil {
					uiErr = fmt.Errorf(T("err_restore_mkdir"), err)
				} else if err := moveDirContents(p.Target, p.Link); err != nil {
					uiErr = fmt.Errorf(T("err_restore"), err)
				} else if err := os.RemoveAll(p.Target); err != nil {
					// Files already restored — still remove DB entry, just warn
					cleanWarn = fmt.Errorf(T("err_clean_storage"), err)
				}
			}

			fyne.Do(func() {
				prog.Hide()
				if uiErr != nil {
					dialog.ShowError(uiErr, a.window)
					return
				}
				if cleanWarn != nil {
					dialog.ShowError(cleanWarn, a.window)
				}
				a.removeFromDB(p)
			})
		}()
	}

	if a.app.Preferences().BoolWithFallback("confirm_delete", true) {
		msg := fmt.Sprintf(T("confirm_restore_msg"), p.Link)
		dialog.ShowConfirm(T("confirm_title"), msg, func(yes bool) {
			if yes {
				process()
			}
		}, a.window)
	} else {
		process()
	}
}

// unbindSave removes the junction link and DB entry but leaves files in storage
func (a *SavesBinderApp) unbindSave(p LinkPair) {
	process := func() {
		go func() {
			var uiErr error
			if err := os.Remove(p.Link); err != nil && !os.IsNotExist(err) {
				uiErr = fmt.Errorf(T("err_delete"), err)
			}
			fyne.Do(func() {
				if uiErr != nil {
					dialog.ShowError(uiErr, a.window)
					return
				}
				a.removeFromDB(p)
			})
		}()
	}

	if a.app.Preferences().BoolWithFallback("confirm_delete", true) {
		msg := fmt.Sprintf(T("confirm_unbind_msg"), p.Link)
		dialog.ShowConfirm(T("confirm_title"), msg, func(yes bool) {
			if yes {
				process()
			}
		}, a.window)
	} else {
		process()
	}
}

// destroySave strictly deletes everything: the link, the files in storage, and DB entry
func (a *SavesBinderApp) destroySave(p LinkPair) {
	process := func() {
		prog := dialog.NewProgressInfinite(T("progress_title"), T("progress_destroy"), a.window)
		prog.Show()

		go func() {
			_ = os.Remove(p.Link)

			var uiErr error
			if err := os.RemoveAll(p.Target); err != nil {
				uiErr = fmt.Errorf(T("err_destroy"), err)
			}
			fyne.Do(func() {
				prog.Hide()
				if uiErr != nil {
					dialog.ShowError(uiErr, a.window)
					return
				}
				a.removeFromDB(p)
			})
		}()
	}

	if a.app.Preferences().BoolWithFallback("confirm_delete", true) {
		msg := fmt.Sprintf(T("confirm_destroy_msg"), p.Link)
		dialog.ShowConfirm(T("confirm_title"), msg, func(yes bool) {
			if yes {
				process()
			}
		}, a.window)
	} else {
		process()
	}
}

// bindSave handles file migration and symlink creation
func (a *SavesBinderApp) bindSave() {
	storageDir := strings.TrimSpace(a.storageEntry.Text)
	if storageDir == "" {
		dialog.ShowError(fmt.Errorf(T("err_no_storage")), a.window)
		return
	}

	originalPath := strings.TrimSpace(a.toBindEntry.Text)
	if originalPath == "" {
		dialog.ShowError(fmt.Errorf(T("err_no_saves")), a.window)
		return
	}

	originalAbs, err := filepath.Abs(originalPath)
	if err != nil {
		dialog.ShowError(fmt.Errorf(T("err_path_process"), err), a.window)
		return
	}

	fileInfo, err := os.Stat(originalAbs)
	if err != nil || !fileInfo.IsDir() {
		dialog.ShowError(fmt.Errorf(T("err_not_folder")), a.window)
		return
	}

	gameName := filepath.Base(originalAbs)
	if gameName == "." || gameName == "/" || gameName == "\\" {
		dialog.ShowError(fmt.Errorf(T("err_game_name")), a.window)
		return
	}

	newDir := filepath.Join(storageDir, gameName)

	// Guard: storage and original path must not nest
	if isNestedPath(storageDir, originalAbs) {
		dialog.ShowError(fmt.Errorf(T("err_nested_paths")), a.window)
		return
	}

	// Guard: original must not already be a junction/symlink
	if reparse, err := isReparsePoint(originalAbs); err == nil && reparse {
		dialog.ShowError(fmt.Errorf(T("err_already_reparse"), originalAbs), a.window)
		return
	}

	// Guard: path already tracked in DB
	for _, existing := range a.db.Links {
		if strings.EqualFold(filepath.Clean(existing.Link), filepath.Clean(originalAbs)) {
			dialog.ShowError(fmt.Errorf(T("err_already_bound")), a.window)
			return
		}
		if strings.EqualFold(filepath.Clean(existing.Target), filepath.Clean(newDir)) {
			dialog.ShowError(fmt.Errorf(T("err_name_collision"), newDir), a.window)
			return
		}
	}

	if _, err := os.Stat(newDir); !os.IsNotExist(err) {
		dialog.ShowError(fmt.Errorf(T("err_already_exists"), newDir), a.window)
		return
	}

	if err := os.MkdirAll(newDir, os.ModePerm); err != nil {
		dialog.ShowError(fmt.Errorf(T("err_create_storage"), err), a.window)
		return
	}

	prog := dialog.NewProgressInfinite(T("progress_title"), T("progress_bind"), a.window)
	prog.Show()

	// Heavy move + junction creation off the UI thread
	go func(originalAbs, newDir, gameName string) {
		errProcess := func() error {
			if err := moveDirContents(originalAbs, newDir); err != nil {
				return err
			}

			remFiles, err := os.ReadDir(originalAbs)
			if err == nil && len(remFiles) > 0 {
				return fmt.Errorf(T("err_empty_folder"))
			}

			if err := os.Remove(originalAbs); err != nil {
				return err
			}

			cmd := exec.Command("cmd", "/c", "mklink", "/J", originalAbs, newDir)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

			if err := cmd.Run(); err != nil {
				return fmt.Errorf("mklink_error: %w", err)
			}

			return nil
		}()

		if errProcess != nil {
			// Best-effort rollback of partial moves (still off UI thread)
			if strings.Contains(errProcess.Error(), "mklink_error") {
				_ = os.RemoveAll(newDir)
			} else if _, err := os.Stat(newDir); !os.IsNotExist(err) {
				_ = os.MkdirAll(originalAbs, os.ModePerm)
				_ = moveDirContents(newDir, originalAbs)
				_ = os.RemoveAll(newDir)
			}

			fyne.Do(func() {
				prog.Hide()
				if strings.Contains(errProcess.Error(), "mklink_error") {
					dialog.ShowError(fmt.Errorf(T("err_junction")), a.window)
				} else {
					dialog.ShowError(fmt.Errorf(T("err_bind_failed"), errProcess), a.window)
				}
			})
			return
		}

		fyne.Do(func() {
			prog.Hide()
			a.db.Links = append(a.db.Links, LinkPair{
				Target: newDir,
				Link:   originalAbs,
				Type:   "junction",
			})
			a.saveData()
			a.invalidatePathHealth()
			a.refreshList()

			a.toBindEntry.SetText("")
			dialog.ShowInformation(T("success_title"), fmt.Sprintf(T("success_bind"), gameName), a.window)
		})
	}(originalAbs, newDir, gameName)
}
