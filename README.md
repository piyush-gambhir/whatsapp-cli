# whatsapp

A read-only command line for a WhatsApp Business Account, built for agents run
through [Reckon](https://github.com/piyush-gambhir/reckon). It reads phone numbers
(quality rating, messaging limit), message templates and their review status, and
message, conversation and pricing analytics. It cannot send messages or change
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
| `numbers list`, `numbers get <id>` | `GET /<WABA>/phone_numbers`, `GET /<id>` |
| `templates list` | `GET /<WABA>/message_templates` |
| `analytics messages` | `fields=analytics.start().end().granularity(DAY)` |
| `analytics conversations` | `fields=conversation_analytics...` (conversation-based pricing) |
| `analytics pricing` | `fields=pricing_analytics...` (per-message pricing) |

Settings come only from the environment: `WHATSAPP_ACCESS_TOKEN`,
`WHATSAPP_BUSINESS_ACCOUNT_ID`, `WHATSAPP_API_VERSION` (default `v25.0`). The token
is sent in a header, never in a URL. Exit codes: 0 ok, 1 usage or local error, 3 a
Graph API error (its code, subcode and trace id are printed).

Not included yet: template analytics (Meta requires a one-way opt-in `POST` on the
account first), and messaging-limit history. Field names follow Meta's documentation
as of October 2026 and have not yet been run against a live account.

```bash
make test vet build   # ./bin/whatsapp
```
