//go:build linux || darwin || freebsd || openbsd || netbsd

package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func openAndValidateConfig(path string) (*os.File, error) {
	// Open without following symlinks (final path component).
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}

	f := os.NewFile(uintptr(fd), path)
	if f == nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("open config: os.NewFile failed")
	}

	// Validate the opened file (TOCTOU resistant).
	// Note: regular file check already done by ValidateLocalManifest
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		f.Close()
		return nil, fmt.Errorf("fstat config: %w", err)
	}

	uid := uint32(os.Getuid())
	if st.Uid != uid {
		f.Close()
		return nil, fmt.Errorf(
			"config not owned by current user (uid %d != %d)",
			st.Uid,
			uid,
		)
	}

	perm := os.FileMode(st.Mode).Perm()
	if perm&0022 != 0 {
		f.Close()
		return nil, fmt.Errorf("config is group/world writable (mode %o)", perm)
	}

	// Parent directory checks (prevents replace attacks).
	dir := filepath.Dir(path)

	var dst unix.Stat_t
	if err := unix.Stat(dir, &dst); err != nil {
		f.Close()
		return nil, fmt.Errorf("stat parent dir: %w", err)
	}
	if dst.Mode&unix.S_IFMT != unix.S_IFDIR {
		f.Close()
		return nil, fmt.Errorf("parent is not a directory: %s", dir)
	}
	if dst.Uid != uid {
		f.Close()
		return nil, fmt.Errorf(
			"parent dir not owned by current user (dir uid %d != %d)",
			dst.Uid,
			uid,
		)
	}
	dperm := os.FileMode(dst.Mode).Perm()
	if dperm&0022 != 0 {
		f.Close()
		return nil, fmt.Errorf(
			"parent dir is group/world writable (dir %s mode %o)",
			dir,
			dperm,
		)
	}

	return f, nil
}

func writeLocalManifestFile(path string, data []byte, perm os.FileMode) error {
	return writePrivateRuntimeFile(path, data, perm, false)
}

func writePrivateRuntimeFile(path string, data []byte, perm os.FileMode, exclusive bool) error {
	flags := unix.O_WRONLY | unix.O_CREAT | unix.O_NOFOLLOW | unix.O_CLOEXEC
	if exclusive {
		flags |= unix.O_EXCL
	}
	fd, err := unix.Open(path, flags, uint32(perm.Perm()))
	if err != nil {
		return fmt.Errorf("open private file: %w", err)
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return fmt.Errorf("open private file: os.NewFile failed")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("inspect opened private file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("private file must be regular: %s", path)
	}
	if err := file.Chmod(perm); err != nil {
		return fmt.Errorf("restrict private file: %w", err)
	}
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("truncate private file: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write private file: %w", err)
	}
	return file.Close()
}
