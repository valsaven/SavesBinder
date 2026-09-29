package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// cleanupJunction removes the junction directory on rollback, returning any error.
func cleanupJunction(absLink string) error {
	if err := os.Remove(absLink); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean up junction directory: %w", err)
	}
	return nil
}

// createJunction creates an NTFS directory junction at linkPath pointing to targetPath.
// linkPath must not exist; targetPath must be an existing directory.
// Uses FSCTL_SET_REPARSE_POINT (no cmd.exe / mklink).
func createJunction(linkPath, targetPath string) error {
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}
	absLink, err := filepath.Abs(linkPath)
	if err != nil {
		return err
	}

	info, err := os.Stat(absTarget)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("junction target is not a directory: %s", absTarget)
	}

	if err := os.Mkdir(absLink, 0o755); err != nil {
		return err
	}

	linkPtr, err := windows.UTF16PtrFromString(absLink)
	if err != nil {
		if cleanupErr := cleanupJunction(absLink); cleanupErr != nil {
			return fmt.Errorf("%w (cleanup failed: %v)", err, cleanupErr)
		}
		return err
	}

	handle, err := windows.CreateFile(
		linkPtr,
		windows.GENERIC_WRITE,
		0,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		if cleanupErr := cleanupJunction(absLink); cleanupErr != nil {
			return fmt.Errorf("%w (cleanup failed: %v)", err, cleanupErr)
		}
		return err
	}
	defer windows.CloseHandle(handle)

	// NT device path form required for mount-point reparse data
	substituteName := `\??\` + absTarget
	subUTF16, err := windows.UTF16FromString(substituteName)
	if err != nil {
		if cleanupErr := cleanupJunction(absLink); cleanupErr != nil {
			return fmt.Errorf("%w (cleanup failed: %v)", err, cleanupErr)
		}
		return err
	}
	printUTF16, err := windows.UTF16FromString(absTarget)
	if err != nil {
		if cleanupErr := cleanupJunction(absLink); cleanupErr != nil {
			return fmt.Errorf("%w (cleanup failed: %v)", err, cleanupErr)
		}
		return err
	}

	// Lengths are in bytes and exclude the terminating NUL
	subLen := uint16((len(subUTF16) - 1) * 2)
	printLen := uint16((len(printUTF16) - 1) * 2)

	// PathBuffer = SubstituteName\0 PrintName\0
	pathBuf := append(append([]uint16{}, subUTF16...), printUTF16...)
	pathBytes := len(pathBuf) * 2

	// MountPointReparseBuffer fixed fields = 8 bytes, then PathBuffer
	reparseDataLength := uint16(8 + pathBytes)
	buf := make([]byte, 8+int(reparseDataLength))

	binary.LittleEndian.PutUint32(buf[0:], windows.IO_REPARSE_TAG_MOUNT_POINT)
	binary.LittleEndian.PutUint16(buf[4:], reparseDataLength)
	binary.LittleEndian.PutUint16(buf[6:], 0) // Reserved
	binary.LittleEndian.PutUint16(buf[8:], 0) // SubstituteNameOffset
	binary.LittleEndian.PutUint16(buf[10:], subLen)
	// Print name starts after substitute + its NUL
	binary.LittleEndian.PutUint16(buf[12:], subLen+2)
	binary.LittleEndian.PutUint16(buf[14:], printLen)

	for i, v := range pathBuf {
		binary.LittleEndian.PutUint16(buf[16+i*2:], v)
	}

	var bytesReturned uint32
	if err := windows.DeviceIoControl(
		handle,
		windows.FSCTL_SET_REPARSE_POINT,
		&buf[0],
		uint32(len(buf)),
		nil,
		0,
		&bytesReturned,
		nil,
	); err != nil {
		if cleanupErr := cleanupJunction(absLink); cleanupErr != nil {
			return fmt.Errorf("%w (cleanup failed: %v)", err, cleanupErr)
		}
		return err
	}

	return nil
}
