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
	// Note: symlink check already done by ValidateLocalManifest
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

	return f, nil
}

func privateFileACL() (*windows.ACL, error) {
	me, err := currentUserSID()
	if err != nil {
		return nil, err
	}

	admins, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return nil, err
	}
	system, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return nil, err
	}

	access := []windows.EXPLICIT_ACCESS{
		fullControlAccess(me, windows.TRUSTEE_IS_USER),
		fullControlAccess(admins, windows.TRUSTEE_IS_GROUP),
		fullControlAccess(system, windows.TRUSTEE_IS_USER),
	}

	return windows.ACLFromEntries(access, nil)
}

func writeLocalManifestFile(path string, data []byte, perm os.FileMode) error {
	return writePrivateRuntimeFile(path, data, perm, false)
}

func writePrivateRuntimeFile(path string, data []byte, _ os.FileMode, exclusive bool) error {
	acl, err := privateFileACL()
	if err != nil {
		return err
	}
	descriptor, err := windows.NewSecurityDescriptor()
	if err != nil {
		return fmt.Errorf("create private security descriptor: %w", err)
	}
	if err := descriptor.SetDACL(acl, true, false); err != nil {
		return fmt.Errorf("set private DACL: %w", err)
	}
	if err := descriptor.SetControl(windows.SE_DACL_PROTECTED, windows.SE_DACL_PROTECTED); err != nil {
		return fmt.Errorf("protect private DACL: %w", err)
	}
	attributes := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: descriptor}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return fmt.Errorf("encode private path: %w", err)
	}
	creation := uint32(windows.OPEN_ALWAYS)
	if exclusive {
		creation = windows.CREATE_NEW
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_WRITE|windows.WRITE_DAC, 0, &attributes, creation, windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return fmt.Errorf("open private file: %w", err)
	}
	file := os.NewFile(uintptr(handle), path)
	if file == nil {
		_ = windows.CloseHandle(handle)
		return fmt.Errorf("open private file: os.NewFile failed")
	}
	defer file.Close()
	var information windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &information); err != nil {
		return fmt.Errorf("inspect opened private file: %w", err)
	}
	if information.FileAttributes&(windows.FILE_ATTRIBUTE_REPARSE_POINT|windows.FILE_ATTRIBUTE_DIRECTORY) != 0 {
		return fmt.Errorf("private file must be regular: %s", path)
	}
	// Creation uses the protected descriptor. Existing files must be restricted
	// through their opened handle before any old content is truncated or replaced.
	if err := windows.SetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		return fmt.Errorf("secure opened private file: %w", err)
	}
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("truncate private file: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write private file: %w", err)
	}
	return file.Close()
}

func applyPrivateACL(path string) error {
	acl, err := privateFileACL()
	if err != nil {
		return err
	}

	return windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		acl,
		nil,
	)
}

func fullControlAccess(sid *windows.SID, trusteeType windows.TRUSTEE_TYPE) windows.EXPLICIT_ACCESS {
	return windows.EXPLICIT_ACCESS{
		AccessPermissions: windows.ACCESS_MASK(windows.GENERIC_ALL),
		AccessMode:        windows.GRANT_ACCESS,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  trusteeType,
			TrusteeValue: windows.TrusteeValueFromSID(sid),
		},
	}
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
