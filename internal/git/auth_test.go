package git

import (
	"net/http"
	"os/exec"
	"testing"
)

func TestCredentialAuthHostRestriction(t *testing.T) {
	for _, host := range []string{"github.com", "other.example"} {
		t.Run(host, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, "https://"+host+"/repo", nil)
			if err != nil {
				t.Fatal(err)
			}
			auth := credentialAuth{host: "github.com", username: "user", password: "secret"}
			if err := auth.Authorizer(request); err != nil {
				t.Fatal(err)
			}
			user, password, ok := request.BasicAuth()
			if host == "github.com" {
				if !ok || user != "user" || password != "secret" {
					t.Fatal("credentials not applied")
				}
			} else if ok {
				t.Fatal("credentials leaked to another host")
			}
		})
	}
}

func TestRemoteClientOptionsCredentialHelper(t *testing.T) {
	path := t.TempDir()
	if output, err := exec.Command("git", "init", path).CombinedOutput(); err != nil {
		t.Fatalf("initialize credential fixture: %s: %v", output, err)
	}
	helper := "!f() { printf 'username=test-user\\npassword=test-token\\n'; }; f"
	if output, err := exec.Command("git", "-C", path, "config", "credential.helper", helper).CombinedOutput(); err != nil {
		t.Fatalf("configure helper: %s: %v", output, err)
	}
	if options := RemoteClientOptions(path, "https://example.invalid/repo.git"); len(options) != 1 {
		t.Fatalf("credential helper did not supply authentication: %d options", len(options))
	}
	for _, remote := range []string{"git@example.invalid:repo.git", path, "https://example.invalid/repo%0Ahost=other.invalid"} {
		if options := RemoteClientOptions(path, remote); len(options) != 0 {
			t.Fatalf("unexpected authentication for %q", remote)
		}
	}
}
