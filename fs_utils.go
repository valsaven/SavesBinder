package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
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

	// Create destination file for writing — fail if already exists to prevent silent overwrites
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, srcInfo.Mode())
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

// backupDir creates a backup of src directory inside backupRoot with a timestamp suffix.
// Returns the path to the created backup directory.
func backupDir(src, backupRoot string) (string, error) {
	if err := os.MkdirAll(backupRoot, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create backup root: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	baseName := filepath.Base(src)
	backupPath := filepath.Join(backupRoot, baseName+"_"+timestamp)

	if err := copyFileOrDir(src, backupPath); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	return backupPath, nil
}

// verifyMoveIntegrity checks that the number of files and total size match between src and dst.
func verifyMoveIntegrity(src, dst string) error {
	srcCount, srcSize, err := countFilesAndSize(src)
	if err != nil {
		return fmt.Errorf("failed to count source files: %w", err)
	}

	dstCount, dstSize, err := countFilesAndSize(dst)
	if err != nil {
		return fmt.Errorf("failed to count destination files: %w", err)
	}

	if srcCount != dstCount {
		return fmt.Errorf("file count mismatch: src=%d, dst=%d", srcCount, dstCount)
	}
	if srcSize != dstSize {
		return fmt.Errorf("size mismatch: src=%d, dst=%d", srcSize, dstSize)
	}

	return nil
}

// countFilesAndSize returns the number of files and total size in bytes under root.
func countFilesAndSize(root string) (int, int64, error) {
	var count int
	var size int64

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
			size += info.Size()
		}
		return nil
	})

	return count, size, err
}

// moveDirContentsSafe moves directory contents with backup and integrity verification.
// On failure, it attempts to restore the original state from the backup.
// Returns the backup path (for cleanup) and any error encountered.
func moveDirContentsSafe(src, dst, backupRoot string) (string, error) {
	// 1. Create backup before any modifications
	backupPath, err := backupDir(src, backupRoot)
	if err != nil {
		return "", fmt.Errorf("backup failed: %w", err)
	}

	// 2. Copy files to destination
	if err := copyFileOrDir(src, dst); err != nil {
		_ = os.RemoveAll(backupPath)
		return "", fmt.Errorf("copy failed: %w", err)
	}

	// 3. Verify integrity
	if err := verifyMoveIntegrity(src, dst); err != nil {
		_ = os.RemoveAll(dst)
		_ = os.RemoveAll(backupPath)
		return "", fmt.Errorf("integrity check failed: %w", err)
	}

	// 4. Remove source files
	if err := os.RemoveAll(src); err != nil {
		// Attempt to restore from backup
		_ = os.MkdirAll(src, os.ModePerm)
		_ = copyFileOrDir(backupPath, src)
		_ = os.RemoveAll(dst)
		_ = os.RemoveAll(backupPath)
		return "", fmt.Errorf("failed to remove source: %w", err)
	}

	return backupPath, nil
}
