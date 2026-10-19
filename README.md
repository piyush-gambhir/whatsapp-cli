# whatsapp

A read-only command line for a WhatsApp Business Account, built for agents run
through [Reckon](https://github.com/piyush-gambhir/reckon). It reads phone numbers
(quality rating, messaging limit), message templates and their review status, and
message and pricing analytics. It cannot send messages or change
anything: every request is a `GET` to the Graph API.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/piyush-gambhir/whatsapp-cli/main/install.sh | sh
```

Installs to `~/.local/bin` by default. Override with `INSTALL_DIR` or pin a release with `VERSION`:

```bash
curl -fsSL https://raw.githubusercontent.com/piyush-gambhir/whatsapp-cli/main/install.sh | VERSION=v0.1.0 sh
```

The installer supports macOS/Linux amd64 and arm64 and verifies SHA-256 checksums. Prebuilt macOS/Linux/Windows archives (`whatsapp-cli_<os>_<arch>.tar.gz`, `.zip` on Windows) and `checksums.txt` are on the [releases page](https://github.com/piyush-gambhir/whatsapp-cli/releases); Windows archives contain `whatsapp.exe`.

From source (Go 1.27+):

```bash
git clone https://github.com/piyush-gambhir/whatsapp-cli.git
cd whatsapp-cli
make build      # cli-go/bin/whatsapp
make install    # $(go env GOPATH)/bin/whatsapp
```

`whatsapp version` prints the version, commit and build date as JSON.

## Usage

```bash
export WHATSAPP_ACCESS_TOKEN=...            # system user token with whatsapp_business_management
export WHATSAPP_BUSINESS_ACCOUNT_ID=1029... # the WABA id
whatsapp numbers list
whatsapp templates list --status rejected
whatsapp analytics pricing --from 2026-10-01 --to 2026-10-08 --dimension pricing_category --dimension country
whatsapp analytics messages --granularity DAY
whatsapp commands resolve -- numbers list    # agent-cli/v1: what would run, offline
```

| Command | Graph API |
| --- | --- |
| `account get` | `GET /<WABA>?fields=id,name,currency,timezone_id,...` |
| `numbers list [--after]`, `numbers get <id>` | `GET /<WABA>/phone_numbers`, `GET /<id>` |
| `templates list [--status] [--after]` | `GET /<WABA>/message_templates` |
| `analytics messages` | `fields=analytics.start().end().granularity(DAY)` |
| `analytics pricing` | `fields=pricing_analytics...` (per-message pricing) |

Settings come only from the environment: `WHATSAPP_ACCESS_TOKEN`,
`WHATSAPP_BUSINESS_ACCOUNT_ID`, `WHATSAPP_API_VERSION` (default `v25.0`, at least
`v24.0`). The token is sent in a header, never in a URL, and the `paging.next` and
`paging.previous` URLs (which can carry it) are dropped from output; page with
`--after <paging.cursors.after>`. Exit codes: 0 ok, 1 usage or local error, 3 a Graph
API error (its code, subcode and trace id are printed), 4 the API could not be reached.

Analytics take `--from`/`--to` as `YYYY-MM-DD` (UTC; `--to` exclusive) or 10-digit unix
seconds, within the last year. Filters are checked before any request: phone numbers
(normalised to digits), 2-letter country codes, and the dimensions each metric
accepts.

Not included: conversation analytics (conversation-based pricing ended on
2025-07-01 and Meta keeps a year of data, so there is nothing left to read), and
template analytics (Meta requires a one-way opt-in `POST` on the
account first), and messaging-limit history. Field names follow Meta's documentation
as of October 2026 and have not yet been run against a live account.

## Agents

The agent skill is [`whatsapp/SKILL.md`](whatsapp/SKILL.md), and every command and flag is in [`docs/commands.md`](docs/commands.md) (generated with `make docs`). `whatsapp commands resolve -- <arguments>` shows offline, without credentials, which command, effect (always `read`) and flags the arguments would run.

## Development

```bash
make test vet build   # cli-go/bin/whatsapp
```
