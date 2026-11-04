package git

import (
	"io"
	"net/http"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing/client"
	"github.com/go-git/go-git/v6/storage/memory"
)

type credentialRedirectTransport struct {
	credentialReachedChild bool
	calls                  int
}

func (rt *credentialRedirectTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	rt.calls++
	if request.URL.Host == "parent.invalid" {
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"https://child.parent.invalid/repo/info/refs?service=git-upload-pack"}}, Body: io.NopCloser(strings.NewReader("")), Request: request}, nil
	}
	rt.credentialReachedChild = request.Header.Get("Authorization") != ""
	return &http.Response{StatusCode: 401, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: request}, nil
}

func TestCredentialsDoNotFollowSubdomainRedirect(t *testing.T) {
	rt := &credentialRedirectTransport{}
	remote := gogit.NewRemote(memory.NewStorage(), &config.RemoteConfig{Name: "origin", URLs: []string{"https://parent.invalid/repo"}})
	_, err := remote.List(&gogit.ListOptions{ClientOptions: []client.Option{
		client.WithHTTPClient(&http.Client{Transport: scopedCredentialTransport{base: rt, auth: credentialAuth{host: "parent.invalid", username: "fake-user", password: "synthetic-test-value"}}}),
	}})
	t.Logf("transport calls=%d; credential reached different host=%v; final error=%v", rt.calls, rt.credentialReachedChild, err)
	if rt.calls != 2 {
		t.Fatal("fixture did not exercise redirect")
	}
	if rt.credentialReachedChild {
		t.Fatal("credential restricted to parent.invalid reached child.parent.invalid")
	}
}
