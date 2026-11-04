package git

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/client"
	"github.com/go-git/go-git/v6/plumbing/format/pktline"
)

func TestEmptyHTTPSClonePreservesInitialBranchInOneDiscovery(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		user, password, authenticated := request.BasicAuth()
		if !authenticated || user != "test-user" || password != "test-password" {
			t.Error("clone transport lost configured authentication")
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch request.Method {
		case http.MethodGet:
			writer.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
			for _, line := range []string{"version 2\n", "agent=test\n", "ls-refs=unborn\n", "object-format=sha1\n"} {
				pktline.WriteString(writer, line)
			}
			pktline.WriteFlush(writer)
		case http.MethodPost:
			body, err := io.ReadAll(request.Body)
			if err != nil || !strings.Contains(string(body), "command=ls-refs") || !strings.Contains(string(body), "unborn\n") {
				t.Errorf("unexpected empty-clone discovery request %q %v", body, err)
			}
			writer.Header().Set("Content-Type", "application/x-git-upload-pack-result")
			pktline.WriteString(writer, "unborn HEAD symref-target:refs/heads/main\n")
			pktline.WriteFlush(writer)
		default:
			t.Errorf("unexpected method %s", request.Method)
		}
	}))
	defer server.Close()
	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	repo, err := clonePreservingUnbornHead(filepath.Join(t.TempDir(), "clone"), server.URL+"/dotfiles.git", []client.Option{
		client.WithHTTPClient(server.Client()),
		client.WithHTTPAuth(credentialAuth{host: endpoint.Host, username: "test-user", password: "test-password"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	head, err := repo.Reference(plumbing.HEAD, false)
	if err != nil || head.Target() != plumbing.NewBranchReferenceName("main") {
		t.Fatalf("initial clone HEAD %v %v", head, err)
	}
	if count := requests.Load(); count != 2 {
		t.Fatalf("clone issued %d requests; want one advertisement and one reference discovery", count)
	}
}
