# Security policy

Security fixes target the latest release. Update before reporting a potential issue.

Please report vulnerabilities privately using [GitHub private vulnerability reporting](https://github.com/piyush-gambhir/whatsapp-cli/security/advisories/new). Do not include real access tokens, phone numbers, message content, or other private data in a public issue.

This CLI stores nothing locally. The access token is read from `WHATSAPP_ACCESS_TOKEN`, sent only in an `Authorization` header (never in a URL), and kept out of error messages and output. The `paging.next` and `paging.previous` URLs in Graph responses can carry a token, so they are removed from output.

Every operation is a read: each request is a `GET` to the WhatsApp Business Management API on the Graph API. The CLI cannot send messages or change templates, numbers, or the account. Use a system user token with only the permissions you need (`whatsapp_business_management`).

The CLI makes no background requests and has no update check. GitHub is contacted only by `install.sh`. Release checksums detect corruption or mismatched downloads; they are not a separate publisher signature.

Meta controls the Graph API. Report WhatsApp platform vulnerabilities through Meta's security reporting process rather than this repository.
