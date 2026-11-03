package git

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/client"
)

// credentialAuth restricts credentials to the host that requested them.
type credentialAuth struct {
	host, username, password string
}

func (a credentialAuth) Authorizer(request *http.Request) error {
	if request.URL.Host == a.host {
		request.SetBasicAuth(a.username, a.password)
	}
	return nil
}

// preparedCredentialOptions holds init's one-use credential lookup.
var preparedCredentialOptions = struct {
	sync.Mutex
	values map[string][]client.Option
}{values: make(map[string][]client.Option)}

// PrepareRemoteCredentials checks cached helper credentials once during init.
// The next transport operation consumes the result instead of querying again.
func PrepareRemoteCredentials(repoPath, remoteURL string) bool {
	options, available := lookupRemoteClientOptions(repoPath, remoteURL)
	preparedCredentialOptions.Lock()
	preparedCredentialOptions.values[remoteURL] = options
	preparedCredentialOptions.Unlock()
	return available
}

// RemoteClientOptions resolves HTTPS credentials through Git's configured helpers.
// This includes Git Credential Manager and gh auth setup-git.
func RemoteClientOptions(repoPath, remoteURL string) []client.Option {
	preparedCredentialOptions.Lock()
	options, prepared := preparedCredentialOptions.values[remoteURL]
	delete(preparedCredentialOptions.values, remoteURL)
	preparedCredentialOptions.Unlock()
	if prepared {
		return options
	}
	options, _ = lookupRemoteClientOptions(repoPath, remoteURL)
	return options
}

func lookupRemoteClientOptions(repoPath, remoteURL string) ([]client.Option, bool) {
	endpoint, err := url.Parse(remoteURL)
	if err != nil || endpoint.Scheme != "https" || endpoint.Host == "" || endpoint.User != nil || strings.ContainsAny(endpoint.Host+endpoint.Path, "\r\n") {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "git", "credential", "fill")
	if repoPath != "" {
		command.Dir = repoPath
	}
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=never")
	command.Stdin = strings.NewReader("protocol=https\nhost=" + endpoint.Host + "\npath=" + strings.TrimPrefix(endpoint.Path, "/") + "\n\n")
	output, err := command.Output()
	if err != nil {
		// Anonymous access may still work for public repositories. Preserve
		// the lookup failure only if the server subsequently requires auth.
		return credentialLookupDiagnostic(err), false
	}
	auth := credentialAuth{host: endpoint.Host}
	for _, line := range strings.Split(string(output), "\n") {
		key, value, _ := strings.Cut(strings.TrimSuffix(line, "\r"), "=")
		switch key {
		case "username":
			auth.username = value
		case "password":
			auth.password = value
		}
	}
	if auth.password == "" {
		return credentialLookupDiagnostic(fmt.Errorf("helper returned no password or token")), false
	}
	return []client.Option{client.WithHTTPAuth(auth)}, true
}

func repositoryClientOptions(repo *git.Repository) []client.Option {
	remote, err := repo.Remote("origin")
	if err != nil || len(remote.Config().URLs) == 0 {
		return nil
	}
	path := ""
	if worktree, err := repo.Worktree(); err == nil {
		path = worktree.Filesystem.Root()
	}
	return RemoteClientOptions(path, remote.Config().URLs[0])
}

// credentialLookupTransport allows public access while explaining why a private
// remote was contacted without credentials. Helper stderr/output may contain
// secrets, so only its exit error is reported.
type credentialLookupTransport struct {
	base      http.RoundTripper
	lookupErr error
}

func (t credentialLookupTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(request)
	if err == nil && (response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden) {
		response.Body.Close()
		return nil, fmt.Errorf("HTTPS access rejected (HTTP %d); Git credential lookup failed: %w; configure a credential helper or select SSH during init", response.StatusCode, t.lookupErr)
	}
	return response, err
}

func credentialLookupDiagnostic(err error) []client.Option {
	return []client.Option{client.WithHTTPClient(&http.Client{Transport: credentialLookupTransport{base: http.DefaultTransport, lookupErr: err}})}
}

// ForgetPreparedCredentials discards init's helper result after authentication setup.
func ForgetPreparedCredentials(remoteURL string) {
	preparedCredentialOptions.Lock()
	delete(preparedCredentialOptions.values, remoteURL)
	preparedCredentialOptions.Unlock()
}
