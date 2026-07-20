package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// isReparsePoint reports whether path is a Windows junction/symlink (reparse point).
func isReparsePoint(path string) (bool, error) {
	const fileAttributeReparsePoint = 0x400

	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false, err
	}
	attrs, err := syscall.GetFileAttributes(p)
	if err != nil {
		return false, err
	}
	return attrs&fileAttributeReparsePoint != 0, nil
}

// isNestedPath reports whether a and b refer to the same path or one contains the other.
func isNestedPath(a, b string) bool {
	aAbs, errA := filepath.Abs(a)
	bAbs, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return false
	}
	aClean := strings.ToLower(filepath.Clean(aAbs))
	bClean := strings.ToLower(filepath.Clean(bAbs))
	if aClean == bClean {
		return true
	}
	sep := string(os.PathSeparator)
	return strings.HasPrefix(aClean+sep, bClean+sep) || strings.HasPrefix(bClean+sep, aClean+sep)
}

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

// restoreOne recreates a broken junction at p.Link pointing to p.Target.
func restoreOne(p LinkPair) (bool, string) {
	if err := createJunction(p.Link, p.Target); err != nil {
		return false, err.Error()
	}
	return true, ""
}
