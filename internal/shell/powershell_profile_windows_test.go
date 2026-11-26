package shell

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestPowerShellProfileParseFailureThroughShortPath(t *testing.T) {
	testPowerShellProfileEditions(t, func(t *testing.T, executable string) {
		paths := powerShellProfileFixtures(t)
		longPath := paths[1]
		writePowerShellFixture(t, paths[0], "$PSModuleAutoLoadingPreference = 'None'")
		writePowerShellFixture(t, longPath, "if (")
		writePowerShellFixture(t, paths[2], `
if ($PSModuleAutoLoadingPreference -cne 'None') { throw 'autoload preference changed' }
if (Get-Module Microsoft.PowerShell.Management, Microsoft.PowerShell.Utility) {
  throw 'the profiler imported a module'
}
`)

		path, err := windows.UTF16PtrFromString(longPath)
		if err != nil {
			t.Fatal(err)
		}
		size, err := windows.GetShortPathName(path, nil, 0)
		if err != nil {
			t.Fatalf("size short profile path: %v", err)
		}
		buf := make([]uint16, size)
		if _, err := windows.GetShortPathName(path, &buf[0], size); err != nil {
			t.Fatalf("get short profile path: %v", err)
		}
		paths[1] = windows.UTF16ToString(buf)
		if strings.EqualFold(filepath.Base(paths[1]), filepath.Base(longPath)) {
			t.Skip("fixture volume does not provide an 8.3 filename")
		}

		report, err := profilePowerShell(context.Background(), executable, &paths)
		if err == nil || !strings.Contains(err.Error(), paths[1]+": could not parse profile: ") ||
			errors.Is(err, errIncompletePowerShellReport) || len(report.Profiles) != 2 ||
			!slices.Contains(report.Warnings, "PowerShell recorded 1 error(s) while running "+paths[1]+".") {
			t.Fatalf("expected a parse failure at the short profile path and two timings, got %+v, %v", report, err)
		}
	})
}
