# WhatsApp CLI command reference

Generated from the command tree. Run `make docs` to refresh.

Global flags apply to every command.

```text
  -o, --output string   output format: json (default "json")
```

## whatsapp

Read a WhatsApp Business Account: numbers, templates and analytics

Read-only access to the WhatsApp Business Management API.

Settings: WHATSAPP_ACCESS_TOKEN (system user token),
WHATSAPP_BUSINESS_ACCOUNT_ID (the WABA), WHATSAPP_API_VERSION (default v25.0).

```text
whatsapp
```

## whatsapp account

The WhatsApp Business Account

```text
whatsapp account
```

## whatsapp account get

Name, currency, timezone, review status

```text
whatsapp account get
```

## whatsapp analytics

Messages and pricing for a time window

```text
whatsapp analytics
```

## whatsapp analytics messages

Messages sent and delivered

```text
whatsapp analytics messages [flags]
```

```text
      --country strings      limit to these 2-letter country codes (repeatable)
      --from string          start: YYYY-MM-DD (UTC) or a unix time (default 7 days ago)
      --granularity string   HALF_HOUR, or the day and month values this metric takes (default "DAY")
      --phone strings        limit to these phone numbers (repeatable)
      --to string            end: YYYY-MM-DD (UTC, exclusive) or a unix time (default now)
```

## whatsapp analytics pricing

Delivered messages and cost per pricing category (per-message pricing)

```text
whatsapp analytics pricing [flags]
```

```text
      --country strings      limit to these 2-letter country codes (repeatable)
      --dimension strings    break down by: COUNTRY,PHONE,PRICING_CATEGORY,PRICING_TYPE,TIER (repeatable)
      --from string          start: YYYY-MM-DD (UTC) or a unix time (default 7 days ago)
      --granularity string   HALF_HOUR, or the day and month values this metric takes (default "DAILY")
      --phone strings        limit to these phone numbers (repeatable)
      --to string            end: YYYY-MM-DD (UTC, exclusive) or a unix time (default now)
```

## whatsapp commands

Offline command discovery for agents

```text
whatsapp commands
```

## whatsapp commands resolve

Show the command, effect and flags that arguments would run, without running them

```text
whatsapp commands resolve -- <arguments>
```

```bash
  whatsapp commands resolve -- numbers list
```

## whatsapp numbers

Business phone numbers, quality ratings and limits

```text
whatsapp numbers
```

## whatsapp numbers get

One number by its id

```text
whatsapp numbers get <phone-number-id>
```

## whatsapp numbers list

Numbers in the account with quality rating and messaging limit (page with --after)

```text
whatsapp numbers list [flags]
```

```text
      --after string   paging.cursors.after from the previous page
```

## whatsapp templates

Message templates and their review status

```text
whatsapp templates
```

## whatsapp templates list

Templates with status, category, language and quality

```text
whatsapp templates list [flags]
```

```text
      --after string      paging.cursors.after from the previous page
      --category string   MARKETING, UTILITY or AUTHENTICATION
      --limit int         most templates to return (1-500) (default 100)
      --name string       filter by template name (Graph's name filter)
      --status string     review status, e.g. APPROVED, PENDING, REJECTED, PAUSED, DISABLED, IN_APPEAL
```

## whatsapp version

Print build information

```text
whatsapp version
```
