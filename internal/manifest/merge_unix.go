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
