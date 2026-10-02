package git

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
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

// RemoteClientOptions resolves HTTPS credentials through Git's configured helpers.
// This includes Git Credential Manager and gh auth setup-git.
func RemoteClientOptions(repoPath, remoteURL string) []client.Option {
	endpoint, err := url.Parse(remoteURL)
	if err != nil || endpoint.Scheme != "https" || endpoint.Host == "" || endpoint.User != nil || strings.ContainsAny(endpoint.Host+endpoint.Path, "\r\n") {
		return nil
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
		return nil
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
		return nil
	}
	return []client.Option{client.WithHTTPAuth(auth)}
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
