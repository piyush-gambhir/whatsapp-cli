# whatsapp

A read-only command line for a WhatsApp Business Account, built for agents run
through [Reckon](https://github.com/piyush-gambhir/reckon). It reads phone numbers
(quality rating, messaging limit), message templates and their review status, and
message and pricing analytics. It cannot send messages or change
anything: every request is a `GET` to the Graph API.

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

```bash
make test vet build   # ./bin/whatsapp
```
