# WhatsApp CLI agent guide

Use `whatsapp --help` and `docs/commands.md` for current flags. See `whatsapp/SKILL.md` for operational guidance.

- Implementation: Go/Cobra in `cli-go/`; module `github.com/piyush-gambhir/whatsapp-cli/cli-go`.
- `make test`, `make vet`, `make build`, and `make docs` run from the repo root.
- Read-only by design: every request is a `GET` to the Graph API (the tests fail on any other method). Do not add send, template, or account-changing commands.
- Settings come only from the environment: `WHATSAPP_ACCESS_TOKEN`, `WHATSAPP_BUSINESS_ACCOUNT_ID`, `WHATSAPP_API_VERSION` (default in `internal/graph`, at least v24.0), and `WHATSAPP_GRAPH_URL` (tests only). The token goes in a header, never in a URL, and `paging.next`/`paging.previous` are dropped from output because they can carry it.
- Tests use `httptest` servers and never contact Meta. Keep local input validation ahead of any request.
- There is no update notifier and no background request. Releases come from GitHub via `install.sh` and the release archives.
- Preserve stdout as command data and stderr as diagnostics. Exit codes: 0 ok, 1 usage or local error, 3 Graph API error, 4 API unreachable.
- `commands resolve -- <args>` is the offline agent-cli/v1 resolver; every command resolves with effect `read`.
- Document user-visible changes and regenerate command docs (`make docs`) when flags change.
