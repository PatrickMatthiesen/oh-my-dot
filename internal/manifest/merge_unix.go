//go:build unix || linux || darwin || freebsd || openbsd || netbsd

package manifest

import (
	"fmt"
	"os"
	"syscall"
)

// validateLocalManifestPlatform performs Unix-specific security checks
func validateLocalManifestPlatform(path string, info os.FileInfo) error {
	// Get file ownership
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		// Can't get detailed file info, allow but warn
		return nil
	}

	// Check ownership - must be owned by current user
	currentUID := uint32(os.Getuid())
	if stat.Uid != currentUID {
		return fmt.Errorf("file not owned by current user (uid %d != %d)", stat.Uid, currentUID)
	}

	// Check permissions - must not be group or world writable
	perm := info.Mode().Perm()
	if perm&0022 != 0 { // Check group-write (020) or other-write (002) bits
		return fmt.Errorf("file is group or world writable (permissions: %o)", perm)
	}

	return nil
}
