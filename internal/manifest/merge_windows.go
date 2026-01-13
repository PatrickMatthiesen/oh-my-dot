//go:build windows

package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

func openAndValidateConfig(path string) (*os.File, error) {
	// Reject reparse points (symlink/junction/etc.).
	if err := rejectReparsePoint(path); err != nil {
		return nil, err
	}

	h, err := windows.CreateFile(
		windows.StringToUTF16Ptr(path),
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}

	f := os.NewFile(uintptr(h), path)
	if f == nil {
		_ = windows.CloseHandle(h)
		return nil, fmt.Errorf("open config: os.NewFile failed")
	}

	// Validate file ACLs.
	if err := checkNoWriteForOthers(path); err != nil {
		f.Close()
		return nil, err
	}

	// Parent directory checks as well.
	dir := filepath.Dir(path)
	if err := rejectReparsePoint(dir); err != nil {
		f.Close()
		return nil, fmt.Errorf("parent dir: %w", err)
	}
	if err := checkNoWriteForOthers(dir); err != nil {
		f.Close()
		return nil, fmt.Errorf("parent dir: %w", err)
	}

	return f, nil
}

func rejectReparsePoint(path string) error {
	attr, err := windows.GetFileAttributes(windows.StringToUTF16Ptr(path))
	if err != nil {
		return fmt.Errorf("get attributes: %w", err)
	}
	if attr&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return fmt.Errorf("file is not a regular file (possibly a symlink)")
	}
	return nil
}

func checkNoWriteForOthers(path string) error {
	me, err := currentUserSID()
	if err != nil {
		return err
	}

	admins, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return err
	}
	system, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return err
	}

	sd, err := windows.GetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		return fmt.Errorf("get security info: %w", err)
	}

	// Extract DACL and owner from security descriptor
	dacl, _, err := sd.DACL()
	if err != nil {
		return fmt.Errorf("get DACL: %w", err)
	}
	if dacl == nil {
		return fmt.Errorf("no DACL present (insecure): %s", path)
	}

	owner, _, err := sd.Owner()
	if err != nil {
		return fmt.Errorf("get owner: %w", err)
	}

	isAllowed := func(sid *windows.SID) bool {
		if sid == nil {
			return false
		}
		if owner != nil && windows.EqualSid(sid, owner) {
			return true
		}
		if me != nil && windows.EqualSid(sid, me) {
			return true
		}
		if windows.EqualSid(sid, admins) {
			return true
		}
		if windows.EqualSid(sid, system) {
			return true
		}
		return false
	}

	// Conservative "dangerous" permissions.
	dangerous := uint32(
		windows.FILE_WRITE_DATA |
			windows.FILE_APPEND_DATA |
			windows.FILE_WRITE_EA |
			windows.FILE_WRITE_ATTRIBUTES |
			windows.DELETE |
			windows.WRITE_DAC |
			windows.WRITE_OWNER |
			windows.GENERIC_WRITE |
			windows.GENERIC_ALL,
	)

	for i := uint16(0); i < dacl.AceCount; i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, uint32(i), &ace); err != nil {
			return fmt.Errorf("get ace: %w", err)
		}
		hdr := &ace.Header

		// Consider only ALLOW ACEs for this audit rule.
		if hdr.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			continue
		}

		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		mask := uint32(ace.Mask)

		if isAllowed(sid) {
			continue
		}
		if mask&dangerous == 0 {
			continue
		}

		sidStr := sid.String()

		// Best-effort friendly name.
		account, domain := "", ""
		if a, d, _, err := sid.LookupAccount(""); err == nil {
			account, domain = a, d
		}

		if account != "" {
			return fmt.Errorf(
				"insecure ACL on %s: %s\\%s has write-like rights (mask 0x%08x)",
				path,
				domain,
				account,
				mask,
			)
		}

		return fmt.Errorf(
			"insecure ACL on %s: SID %s has write-like rights (mask 0x%08x)",
			path,
			sidStr,
			mask,
		)
	}

	return nil
}

func currentUserSID() (*windows.SID, error) {
	tok, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return nil, err
	}
	defer tok.Close()

	u, err := tok.GetTokenUser()
	if err != nil {
		return nil, err
	}
	return u.User.Sid, nil
}
