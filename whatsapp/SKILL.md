---
name: whatsapp
description: Read a WhatsApp Business Account with the whatsapp CLI, covering phone numbers with quality rating and messaging limits, message templates and their review status, and message and pricing analytics. Read-only.
---

# WhatsApp

Use the installed `whatsapp` binary; check subcommand `--help` for flags. The full reference is [commands](../docs/commands.md), with setup in the [README](../README.md).

## When to use

Questions about a WhatsApp Business Account (WABA): which numbers exist and their quality rating or messaging limit, which templates are approved or rejected and why, how many messages were sent and delivered, what per-message pricing cost. It cannot send messages or change anything: every request is a `GET` to the Graph API.

## Setup

Settings come only from the environment. Do not invent credentials or put them in reports.

- `WHATSAPP_ACCESS_TOKEN`: system user token with `whatsapp_business_management`.
- `WHATSAPP_BUSINESS_ACCOUNT_ID`: the numeric WABA id.
- `WHATSAPP_API_VERSION`: optional, default `v25.0`, at least `v24.0`.

## Commands

```bash
whatsapp account get
whatsapp numbers list                       # page with --after <paging.cursors.after>
whatsapp numbers get <phone-number-id>      # the id, not the phone number
whatsapp templates list --status rejected --category utility --limit 50
whatsapp analytics messages --granularity DAY --from 2026-10-01 --to 2026-10-08
whatsapp analytics pricing --from 2026-10-01 --to 2026-10-08 --dimension pricing_category --dimension country
whatsapp version
```

## Output

Output is JSON on stdout (`-o json`, the only format); diagnostics and errors go to stderr. Graph's response is kept as sent, except `paging.next` and `paging.previous`, which can carry the token and are removed. Follow pages with `--after`. Exit codes: 0 ok, 1 usage or local error, 3 Graph API error (code, subcode, trace id printed), 4 API unreachable.

## Analytics

`--from`/`--to` take `YYYY-MM-DD` (UTC, `--to` exclusive) or 10-digit unix seconds, within the last year; default is the last 7 days. `messages` granularity is `HALF_HOUR`, `DAY` or `MONTH`; `pricing` takes `HALF_HOUR`, `DAILY` or `MONTHLY`. Empty results are real: do not invent numbers. Conversation analytics are not available (conversation-based pricing ended 2025-07-01).

## Read-only and agent discovery

All commands read. `whatsapp commands resolve -- numbers list` runs offline with no credentials and prints the command, `effect` (always `read`), flags, and args that the arguments would run, using the real parser. It fails for arguments that name no runnable command.
