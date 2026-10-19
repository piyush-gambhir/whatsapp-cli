# Contributing

Use Go 1.27+ (toolchain Go 1.27.2). From the repository root:

```bash
make test
make vet
make build
make docs
```

Run `gofmt -w .` within `cli-go/` after Go changes. HTTP tests must use `httptest` servers and must not contact Meta. Never commit access tokens, phone numbers, or captured customer data.

Command composition lives in `cli-go/cmd` and the Graph client in `cli-go/internal/graph`. Keep local input validation ahead of any network request, and keep the CLI read-only: no command may issue anything but a `GET`.

Submit changes through a pull request. Commits must be signed. The default branch requires linear history, resolved review threads, and passing Go CI and CodeQL checks; squash or rebase merges are supported.

## Dependency maintenance

Prefer current stable releases, pinned to exact module versions and immutable GitHub Action commit SHAs. Check upstream stable releases when scaffolding or updating dependencies; do not assume another repository's pins are current. Exclude prereleases and review breaking major-version migrations explicitly.

Dependabot checks daily and groups minor/patch updates per ecosystem so coupled packages and CodeQL init/analyze updates are tested together. Major updates remain separate. Run `go get -u -t ./...` and `go mod tidy` inside `cli-go`, then run platform CI. Check the GoReleaser binary version explicitly in both CI and release workflows; Dependabot's GitHub Actions updater does not manage that action input. CI builds snapshot archives using the same GoReleaser version as releases.

To release, run `goreleaser check --config cli-go/.goreleaser.yaml` from the repository root, run CI, set `cli-go/VERSION`, and push a signed version tag. A local packaging check is `goreleaser release --snapshot --clean --skip=publish --config cli-go/.goreleaser.yaml`, also from the root. The release workflow builds binaries and checksums. `install.sh` uses that repository's release assets.
