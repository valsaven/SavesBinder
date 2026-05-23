package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// Helper function to move directory contents
func moveDirContents(src, dst string) error {
	files, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, file := range files {
		srcPath := filepath.Join(src, file.Name())
		dstPath := filepath.Join(dst, file.Name())

		err := os.Rename(srcPath, dstPath)
		if err != nil {
			if err := copyFileOrDir(srcPath, dstPath); err != nil {
				return err
			}

			if err := os.RemoveAll(srcPath); err != nil {
				return fmt.Errorf(T("err_move_cleanup"), err)
			}
		}
	}
	return nil
}

// Helper function to copy files across different logical drives
func copyFileOrDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
			return err
		}
		files, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, file := range files {
			if err := copyFileOrDir(filepath.Join(src, file.Name()), filepath.Join(dst, file.Name())); err != nil {
				return err
			}
		}
		return nil
	}

	// Open source file for reading
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	// Defer is kept as a fallback to ensure the descriptor is closed on early exits, but explicit close at the end handles errors
	defer in.Close()

	// Create destination file for writing
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	// Defer ensures the file handles are released if a panic or unexpected error occurs
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	// Restore original file mode permissions
	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Explicitly close and sync written file to catch flush/disk errors
	if err := out.Close(); err != nil {
		return err
	}

	// Explicitly close input file
	if err := in.Close(); err != nil {
		return err
	}

	return nil
}

// restoreOne executes mklink to restore a broken junction or hardlink
func (a *SavesBinderApp) restoreOne(p LinkPair) (bool, string) {
	var cmd *exec.Cmd
	if p.Type == "hardlink" {
		cmd = exec.Command("cmd", "/c", "mklink", "/H", p.Link, p.Target)
	} else if p.Type == "junction" {
		cmd = exec.Command("cmd", "/c", "mklink", "/J", p.Link, p.Target)
	} else {
		return false, T("unknown_type")
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		return false, err.Error()
	}
	return true, ""
}
