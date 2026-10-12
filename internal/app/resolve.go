package app

import (
	"context"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/lifecycle"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// ResolveOptions are the human's Bootstrap resolution flags.
type ResolveOptions struct {
	Path    string
	Version string
	Offline bool
}

// request turns the flags into a resolution request. With no explicit
// package and no explicit version there is exactly one answer: the release
// this CLI pins. A branch head is never one.
func (o ResolveOptions) request() ports.ResolveRequest {
	req := ports.ResolveRequest{Path: o.Path, Version: o.Version, Offline: o.Offline}
	if req.Path == "" && req.Version == "" {
		req.Version = core.BootstrapPin
	}
	return req
}

// resolve finds, verifies and negotiates a Bootstrap catalog. A catalog that
// fails any of the three is refused before the project is touched.
func (a *App) resolve(ctx context.Context, req ports.ResolveRequest) (ports.Package, error) {
	pkg, err := a.Resolver.Resolve(ctx, req)
	if err != nil {
		return pkg, err
	}
	if err := a.Resolver.Verify(ctx, &pkg); err != nil {
		return pkg, err
	}
	if _, err := a.gate(ctx, a.Factory.At(pkg.Root)); err != nil {
		return pkg, err
	}
	return pkg, nil
}

// catalogFor resolves the catalog matching the project's installed
// Bootstrap: the authorised source of its managed ADE context.
func (a *App) catalogFor(ctx context.Context, p *Project, opts ResolveOptions) (ports.Package, error) {
	st, err := p.Store.Load()
	if err != nil {
		return ports.Package{}, err
	}
	req := ports.ResolveRequest{Path: opts.Path, Offline: opts.Offline, Version: p.Release.Bootstrap.Version}
	if st.Bootstrap != nil {
		req.Recorded = st.Bootstrap.Origin
	}
	return a.resolve(ctx, req)
}

func resolution(pkg ports.Package) lifecycle.Resolution {
	b := pkg.Release.Bootstrap
	return lifecycle.Resolution{ID: b.ID, Version: b.Version, SchemaVersion: b.SchemaVersion, Channel: b.Channel,
		Source: pkg.Source, Origin: pkg.Origin, Checksum: pkg.Checksum, Integrity: pkg.Integrity, Signature: pkg.Signature,
		ResolvedAt: lifecycle.Timestamp()}
}
