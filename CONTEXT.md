# Xero CLI Agent Context

Use these patterns when automating `xero`:

- Prefer `--output json` or `--output jsonl` for machine-readable output.
- Use `xero describe` to inspect commands, flags, env vars, and mutability before invoking a command.
- Use `--fields` to reduce JSON output to the specific paths you need.
- Use `--dry-run` on mutating commands before sending real requests.
- Use `--input` or `--input-file` for structured JSON input on mutating commands.
- Use `--input-file -` to read JSON from stdin.
- Use `--tenant-id` or `XERO_TENANT_ID` for tenant-scoped commands.
- Sensitive fields such as access tokens, refresh tokens, client IDs, and authorization headers are redacted from machine-readable output by default. Use `--redact=false` only if you explicitly need raw values.

Authentication:

- `xero auth login --client-id ... --no-browser`
- `XERO_CLIENT_ID=... xero auth login`
- `xero auth import --client-id ... --access-token ... --refresh-token ...`
- `xero auth status --output json`

Suggested workflow:

1. `xero describe <command>`
2. `xero <command> --dry-run --output json`
3. `xero <command> --output json`
