package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"fyne.io/fyne/v2/dialog"
)

// verifyLinks checks for broken links and prompts for restoration
func (a *SavesBinderApp) verifyLinks() {
	a.refreshList()

	var restorable []LinkPair
	for _, p := range a.db.Links {
		_, errTarget := os.Stat(p.Target)
		_, errLink := os.Stat(p.Link)
		if errTarget == nil && os.IsNotExist(errLink) {
			restorable = append(restorable, p)
		}
	}

	if len(restorable) == 0 {
		dialog.ShowInformation(T("verify_title"), T("verify_ok"), a.window)
		return
	}

	msg := fmt.Sprintf(T("verify_found_msg"), len(restorable))
	dialog.ShowConfirm(T("btn_verify"), msg, func(yes bool) {
		if !yes {
			return
		}

		var failed []string
		for _, p := range restorable {
			success, errStr := a.restoreOne(p)
			if !success {
				failed = append(failed, fmt.Sprintf("%s: %s", p.Link, errStr))
			}
		}

		a.refreshList()

		if len(failed) > 0 {
			errLines := strings.Join(failed, "\n")
			dialog.ShowError(fmt.Errorf(T("verify_restore_err"), errLines), a.window)
		} else {
			dialog.ShowInformation(T("verify_title"), T("verify_ok"), a.window)
		}
	}, a.window)
}

// removeFromDB handles removing the record from JSON and refreshing UI
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
		// 1. Remove junction link safely
		if err := os.Remove(p.Link); err != nil && !os.IsNotExist(err) {
			dialog.ShowError(fmt.Errorf(T("err_delete"), err), a.window)
			return
		}

		// 2. Move files back if target exists
		if _, err := os.Stat(p.Target); err == nil {
			// Check for an error creating the original directory
			if err := os.MkdirAll(p.Link, os.ModePerm); err != nil {
				dialog.ShowError(fmt.Errorf(T("err_restore_mkdir"), err), a.window)
				return
			}

			if err := moveDirContents(p.Target, p.Link); err != nil {
				dialog.ShowError(fmt.Errorf(T("err_restore"), err), a.window)
				return
			}

			// Check for a storage cleanup error (important: we do NOT abort execution via return, as the files have already been successfully transferred to the original! The database still needs to be cleaned up)
			if err := os.RemoveAll(p.Target); err != nil {
				dialog.ShowError(fmt.Errorf(T("err_clean_storage"), err), a.window)
			}
		}

		// 3. Remove from database only after critical operations are done
		a.removeFromDB(p)
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
		if err := os.Remove(p.Link); err != nil && !os.IsNotExist(err) {
			dialog.ShowError(fmt.Errorf(T("err_delete"), err), a.window)
			return
		}
		a.removeFromDB(p)
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
		// Ignore error if link is already missing
		_ = os.Remove(p.Link)

		if err := os.RemoveAll(p.Target); err != nil {
			dialog.ShowError(fmt.Errorf(T("err_destroy"), err), a.window)
			return
		}

		a.removeFromDB(p)
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

	if _, err := os.Stat(newDir); !os.IsNotExist(err) {
		dialog.ShowError(fmt.Errorf(T("err_already_exists"), newDir), a.window)
		return
	}

	if err := os.MkdirAll(newDir, os.ModePerm); err != nil {
		dialog.ShowError(fmt.Errorf(T("err_create_storage"), err), a.window)
		return
	}

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
		if strings.Contains(errProcess.Error(), "mklink_error") {
			dialog.ShowError(fmt.Errorf(T("err_junction")), a.window)
			_ = os.RemoveAll(newDir)
		} else {
			dialog.ShowError(fmt.Errorf(T("err_bind_failed"), errProcess), a.window)
			if _, err := os.Stat(newDir); !os.IsNotExist(err) {
				_ = os.MkdirAll(originalAbs, os.ModePerm)
				_ = moveDirContents(newDir, originalAbs)
				_ = os.RemoveAll(newDir)
			}
		}
		return
	}

	a.db.Links = append(a.db.Links, LinkPair{
		Target: newDir,
		Link:   originalAbs,
		Type:   "junction",
	})
	a.saveData()
	a.invalidatePathHealth()
	a.refreshList()

	a.toBindEntry.SetText("")
	dialog.ShowInformation("Успех", fmt.Sprintf(T("success_bind"), gameName), a.window)
}
