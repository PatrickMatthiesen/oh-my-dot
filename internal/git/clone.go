package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/client"
	"github.com/go-git/go-git/v6/plumbing/protocol/capability"
	"github.com/go-git/go-git/v6/plumbing/protocol/packp"
	"github.com/go-git/go-git/v6/plumbing/transport"
)

// go-git beta.1 discards an empty remote's unborn HEAD while turning its
// advertisement into ErrEmptyRemoteRepository. Preserve it during the same
// discovery request so the first push creates the remote's intended branch.
func clonePreservingUnbornHead(path, remoteURL string, options []client.Option) (*gogit.Repository, error) {
	cloneOptions := &gogit.CloneOptions{URL: remoteURL, AllowEmptyRepo: true}
	if err := cloneOptions.Validate(); err != nil {
		return nil, fmt.Errorf("validate clone options: %w", err)
	}
	endpoint, err := transport.ParseURL(cloneOptions.URL)
	if err != nil {
		return nil, fmt.Errorf("parse clone URL: %w", err)
	}
	base, err := client.New(options...).Transport(endpoint.Scheme)
	if err != nil {
		return nil, fmt.Errorf("prepare clone transport: %w", err)
	}
	tracker := &cloneHeadTransport{Transport: base}
	cloneOptions.ClientOptions = append(options, client.WithTransport(endpoint.Scheme, tracker))
	repo, err := gogit.PlainClone(path, cloneOptions)
	if err != nil {
		return repo, err
	}
	if _, err := repo.Head(); !errors.Is(err, plumbing.ErrReferenceNotFound) {
		if err != nil {
			repo.Close()
			return nil, fmt.Errorf("inspect cloned initial branch: %w", err)
		}
		return repo, nil
	}
	// go-git's in-process file server does not advertise unborn branches.
	// The same local reference can be read directly without another exchange.
	if tracker.unborn == "" && endpoint.Scheme == "file" {
		remote, err := transport.DefaultLoader.Load(endpoint)
		if err != nil {
			repo.Close()
			return nil, fmt.Errorf("inspect empty local remote: %w", err)
		}
		if closer, ok := remote.(io.Closer); ok {
			defer closer.Close()
		}
		if head, err := remote.Reference(plumbing.HEAD); err == nil && head.Type() == plumbing.SymbolicReference {
			tracker.unborn = head.Target()
		}
	}
	if tracker.unborn == "" {
		return repo, nil
	}
	if !tracker.unborn.IsBranch() || tracker.unborn.Validate() != nil {
		repo.Close()
		return nil, fmt.Errorf("remote advertised invalid initial branch %q", tracker.unborn)
	}
	if err := repo.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, tracker.unborn)); err != nil {
		repo.Close()
		return nil, fmt.Errorf("set cloned initial branch: %w", err)
	}
	branch := &config.Branch{Name: tracker.unborn.Short(), Remote: "origin", Merge: tracker.unborn}
	if err := repo.CreateBranch(branch); err != nil {
		repo.Close()
		return nil, fmt.Errorf("configure cloned initial branch: %w", err)
	}
	return repo, nil
}

type cloneHeadTransport struct {
	transport.Transport
	unborn plumbing.ReferenceName
}

func (t *cloneHeadTransport) Handshake(ctx context.Context, request *transport.Request) (transport.Session, error) {
	session, err := t.Transport.Handshake(ctx, request)
	if err != nil {
		return nil, err
	}
	return &cloneHeadSession{Session: session, tracker: t}, nil
}

type cloneHeadSession struct {
	transport.Session
	tracker *cloneHeadTransport
}

func (s *cloneHeadSession) GetRemoteRefs(ctx context.Context, options *transport.GetRemoteRefsOptions) (*transport.RemoteRefs, error) {
	commander, supportsCommands := s.Session.(transport.Commander)
	if !supportsCommands || !slices.Contains(s.Capabilities().Get(capability.LsRefs), "unborn") {
		return s.Session.GetRemoteRefs(ctx, options)
	}
	args := &packp.LsRefsArgs{Peel: true, Symrefs: true, Unborn: true}
	if options != nil {
		args.RefPrefixes = options.RefPrefixes
	}
	output := &packp.LsRefsOutput{}
	if err := commander.Command(ctx, "ls-refs", args, output); err != nil {
		return nil, err
	}
	refs := transport.NewRemoteRefs(output.References)
	s.tracker.unborn = refs.Unborn
	for _, ref := range refs.References {
		if ref.Type() == plumbing.HashReference {
			return refs, nil
		}
	}
	return nil, transport.ErrEmptyRemoteRepository
}
