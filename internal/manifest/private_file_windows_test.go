//go:build windows

package manifest

import (
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestPrivateFileHasProtectedACLOnCreationAndReplacement(t *testing.T) {
	for _, existing := range []bool{false, true} {
		name := "created"
		if existing {
			name = "replaced"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "private-runtime")
			if existing {
				if err := os.WriteFile(path, []byte("public"), 0644); err != nil {
					t.Fatal(err)
				}
				world, err := windows.CreateWellKnownSid(windows.WinWorldSid)
				if err != nil {
					t.Fatal(err)
				}
				acl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{fullControlAccess(world, windows.TRUSTEE_IS_GROUP)}, nil)
				if err != nil {
					t.Fatal(err)
				}
				if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
					t.Fatal(err)
				}
			}
			if err := WritePrivateFile(path, []byte("private")); err != nil {
				t.Fatal(err)
			}
			descriptor, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
			if err != nil {
				t.Fatal(err)
			}
			control, _, err := descriptor.Control()
			if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
				t.Fatalf("private DACL inherits permissions: %v, %v", control, err)
			}
			acl, _, err := descriptor.DACL()
			if err != nil || acl == nil {
				t.Fatalf("private DACL missing: %v", err)
			}
			user, err := currentUserSID()
			if err != nil {
				t.Fatal(err)
			}
			admins, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
			if err != nil {
				t.Fatal(err)
			}
			system, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
			if err != nil {
				t.Fatal(err)
			}
			for index := uint16(0); index < acl.AceCount; index++ {
				var ace *windows.ACCESS_ALLOWED_ACE
				if err := windows.GetAce(acl, uint32(index), &ace); err != nil {
					t.Fatal(err)
				}
				if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
					continue
				}
				sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
				if !windows.EqualSid(sid, user) && !windows.EqualSid(sid, admins) && !windows.EqualSid(sid, system) {
					t.Errorf("private runtime grants another identity access: %s", sid.String())
				}
			}
		})
	}
}
