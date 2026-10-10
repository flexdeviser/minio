# MinIO Fork Release Guide

This document describes how this fork (`flexdeviser/minio`) is versioned and how to cut a release. Upstream MinIO moved to maintenance mode (source-only), so this fork maintains its own release pipeline against GitHub Releases and GHCR.

## Version scheme

This fork keeps the upstream MinIO release-tag format so all existing parsing logic (`cmd/update.go`, `mc admin update`, helm, docs) keeps working:

```
RELEASE.<YYYY>-<MM>-<DD>T<HH>-<MM>-<SS>Z
```

Example: `RELEASE.2026-10-10T12-00-00Z`

The current version is derived from the nearest git tag via `git describe --tags` and injected at build time by `buildscripts/gen-ldflags.go` into `cmd.Version` / `cmd.ReleaseTag`.

## Release process

### 1. Prepare the branch

Make sure `master` is green and the changes you want to ship are merged. Work and fix bugs on `master`, then cut a tag.

### 2. Cut a tag

Tags do not have to be created manually with exact timestamps - the canonical way is to tag the SHA at the UTC time of the commit:

```sh
git checkout master
git pull origin master
# Find the UTC commit time of the tip of master
COMMIT_TIME=$(git log -1 --format=%cI)
# Convert RFC3339 (2026-10-10T12:00:00Z) -> tag format (2026-10-10T12-00-00Z)
TAG="RELEASE.$(echo "$COMMIT_TIME" | sed -E 's/^([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}):([0-9]{2}):([0-9]{2})Z$/\1-\2-\3Z/')"
git tag -a "$TAG" -m "Release $TAG"
git push origin "$TAG"
```

### 3. Pipeline runs automatically

Pushing a tag matching `RELEASE.*` triggers `.github/workflows/release.yml`:

- `version` job resolves the tag into an RFC3339 version and validates it.
- `binaries` job cross-compiles `minio.<tag>.<os>-<arch>` (+ `.sha256sum`) for `linux/{amd64,arm64,ppc64le,s390x}`, `darwin/{amd64,arm64}` and `windows/amd64`.
- `release` job uploads the binaries to a GitHub Release with auto-generated notes.
- `docker` job builds and pushes multi-arch images to:
  - `ghcr.io/flexdeviser/minio:<tag>`
  - `ghcr.io/flexdeviser/minio:latest`

No minisign signing is performed yet (upstream's private key is not available); release integrity is guaranteed by the `.sha256sum` files shipped alongside the binaries and the Git tag itself.

### 4. Verify

```sh
# Confirm the release tag is what is baked in
MINIO_RELEASE=RELEASE go run buildscripts/gen-ldflags.go "$TAG_RFC3339_VERSION"

# Binary
./minio --version
# Image
docker run --rm ghcr.io/flexdeviser/minio:<tag> --version
```

## CVE / dependency maintenance

Run `govulncheck ./...` before each release (installed via `go install golang.org/x/vuln/cmd/govulncheck@latest`).

**Current baseline (as of the RELEASE.2026-02-12T20-18-48Z baseline tag):** `govulncheck` reports 46 reachable vulnerabilities across 9 modules plus the Go standard library:

- Go stdlib (fixed by Go 1.27.2): `crypto/tls`, `net/http`, `html/template`, `net/textproto`
- `golang.org/x/crypto`, `golang.org/x/net`, `golang.org/x/text`
- `github.com/eclipse/paho.mqtt.golang`
- `github.com/go-jose/go-jose/v4`
- `github.com/prometheus/prometheus`
- `github.com/rabbitmq/amqp091-go`
- `go.opentelemetry.io/otel/sdk`
- `google.golang.org/grpc`

### Upgrade procedure (do it incrementally, one commit per module)

Dependency bumps must be done **one module at a time**, verifying each step or they will not converge:

```sh
go get <module>@<fixed-version>
go mod tidy -compat=1.21
go build ./cmd/... ./internal/...
go test -count=1 -tags kqueue,dev ./cmd/...   # relevant packages
govulncheck ./...
```

Known trap: `github.com/prometheus/prometheus` upgrades drag in
`github.com/go-openapi/swag@v0.25.x`, whose test-only dependency chain
(`go-openapi/testify`) fails `go mod tidy`. If a prometheus bump breaks tidy,
pin `go-openapi/swag` back (`go get github.com/go-openapi/swag@v0.23.1`) or
investigate the go-openapi/testify resolution separately. The Go stdlib CVEs
are resolved by moving the CI/runner Go toolchain to 1.27.2+ (see
`.github/workflows/*.yml` `go-version`).

## Notes

- **Update checks disabled by default:** This fork defaults `MINIO_UPDATE=off` (see `cmd/common-main.go`), so the server no longer phones home to `dl.min.io` and `mc admin update` in-place upgrades are disabled. Set `MINIO_UPDATE=on` to re-enable if ever needed.
- **Upstream tracking:** `upstream` remote points at `https://github.com/minio/minio`. Upstream is in maintenance mode, but if a security fix lands there you can cherry-pick it: `git fetch upstream && git cherry-pick <sha>`.
- **Dependency/CVE hygiene:** Dependabot runs weekly for `gomod`, `github-actions` and `docker`. `govulncheck` runs on every PR/push to `master` plus a weekly scheduled run (`.github/workflows/vulncheck.yml`). Before every release, run `govulncheck ./...` locally.
- **License:** AGPLv3. Internal/private use has no extra obligations; if you distribute this software externally you must also make this fork's source code available under AGPLv3.