# Xero CLI

A command-line interface for interacting with the Xero API, modeled after the GitHub CLI (`gh`). Manage your Xero accounting data directly from your terminal with OAuth 2.0 authentication and an intuitive command structure.

## Features

- **OAuth 2.0 Authentication** with PKCE (Proof Key for Code Exchange) for enhanced security
- **Automatic Token Refresh** - No need to re-authenticate frequently
- **Secure Credential Storage** - Credentials stored locally with proper permissions
- **Invoice Management** - List, filter, and manage invoices
- **Clean CLI Interface** - Inspired by GitHub CLI's user-friendly design
- **No External Dependencies** - Uses Erlang's built-in HTTP client and SSL

## Prerequisites

- Elixir 1.14 or higher
- Erlang/OTP 25 or higher
- A Xero developer account and OAuth 2.0 application

## Installation

### 1. Clone and Build

```bash
git clone <repository-url>
cd xero-cli
mix deps.get
mix escript.build
```

This will create an executable file named `xero` in the current directory.

### 2. Make it Available System-Wide (Optional)

```bash
# Linux/macOS
sudo cp xero /usr/local/bin/
chmod +x /usr/local/bin/xero

# Or add to your PATH
export PATH="$PATH:/path/to/xero-cli"
```

## Setup

### 1. Create a Xero OAuth 2.0 Application

1. Go to [Xero Developer Portal](https://developer.xero.com/app/manage)
2. Click "New app" or select an existing app
3. Note your **Client ID**
4. Add `http://localhost:8888/callback` as a redirect URI
5. Enable the required scopes:
   - `openid`
   - `profile`
   - `email`
   - `offline_access`
   - `accounting.transactions`
   - `accounting.contacts`
   - `accounting.settings`

### 2. Authenticate

Run the login command and follow the prompts:

```bash
./xero auth login
```

You'll be asked to:
1. Enter your Xero Client ID
2. Authorize the application in your browser
3. The CLI will automatically capture the OAuth callback

Your credentials will be securely stored in `~/.xero-cli/config.json` with restricted file permissions (600).

## Usage

### Authentication Commands

#### Login to Xero
```bash
xero auth login
```

Starts the OAuth 2.0 authentication flow. Opens your browser automatically and waits for authorization.

#### Check Authentication Status
```bash
xero auth status
```

Shows your current authentication status, organization, and token validity.

#### Logout
```bash
xero auth logout
```

Removes stored credentials.

---

### Invoice Commands

#### List All Invoices
```bash
xero invoices list
```

Displays all invoices in a formatted table with:
- Invoice Number
- Type (ACCREC/ACCPAY)
- Contact Name
- Date
- Due Date
- Status
- Total Amount

#### Filter Invoices by Status
```bash
xero invoices list --status PAID
xero invoices list --status AUTHORISED
xero invoices list --status DRAFT
```

#### Pagination
```bash
xero invoices list --page 2 --page-size 50
```

---

## Examples

### Complete Workflow

```bash
# 1. Authenticate
$ xero auth login
🔐 Xero CLI - OAuth 2.0 Authentication

Enter your Xero Client ID: YOUR_CLIENT_ID_HERE
Please visit the following URL to authorize this application:
  https://login.xero.com/identity/connect/authorize?...

✓ Browser opened automatically
📡 Waiting for OAuth callback on http://localhost:8888/callback...
✓ Authorization code received
Exchanging code for access token...
✓ Access token obtained
Fetching Xero organizations...

✅ Successfully authenticated with Xero!
Organization: My Company Ltd
Tenant ID: abc123-def456-...

You can now use the Xero CLI.

# 2. Check status
$ xero auth status
✅ Authenticated
Organization: My Company Ltd
Tenant ID: abc123-def456-...
✓ Access token is valid

# 3. List invoices
$ xero invoices list
📄 Fetching invoices...

Found 15 invoice(s):

========================================================================================================================
Invoice Number       | Type       | Contact                   | Date         | Due Date     | Status       | Total
========================================================================================================================
INV-001             | ACCREC     | Acme Corp                 | 2025-01-15   | 2025-02-14   | ✓ PAID      | $1,250.00
INV-002             | ACCREC     | Widget Industries         | 2025-01-20   | 2025-02-19   | ○ AUTH      | $3,450.00
INV-003             | ACCREC     | Tech Solutions LLC        | 2025-01-25   | 2025-02-24   | ◐ DRAFT     | $890.50
========================================================================================================================

# 4. Filter invoices
$ xero invoices list --status AUTHORISED
```

## Configuration

Configuration and credentials are stored in `~/.xero-cli/config.json`.

You can also set your Client ID as an environment variable:

```bash
export XERO_CLIENT_ID="your_client_id_here"
xero auth login
```

## Project Structure

```
xero-cli/
├── lib/
│   ├── xero_cli/
│   │   ├── cli.ex              # Main CLI entry point and command router
│   │   ├── config.ex           # Configuration and credential management
│   │   ├── http.ex             # HTTP client wrapper (uses :httpc)
│   │   ├── json.ex             # JSON encoder/decoder
│   │   ├── oauth.ex            # OAuth 2.0 flow implementation
│   │   └── commands/
│   │       ├── auth.ex         # Authentication commands
│   │       └── invoices.ex     # Invoice management commands
│   └── xero_cli.ex             # Main module
├── mix.exs                      # Project configuration
└── README.md                    # This file
```

## Architecture

### OAuth 2.0 Flow

1. **Authorization Request**: Generates a PKCE code verifier and challenge
2. **User Authorization**: Opens browser for user to authorize
3. **Callback Server**: Starts a local server on port 8888 to receive the callback
4. **Token Exchange**: Exchanges authorization code for access and refresh tokens
5. **Token Storage**: Securely stores tokens with proper file permissions
6. **Automatic Refresh**: Refreshes tokens automatically when expired

### Security Features

- **PKCE**: Uses Proof Key for Code Exchange for enhanced OAuth security
- **Token Expiration**: Tracks token expiration and refreshes automatically
- **Secure Storage**: Config file has 600 permissions (owner read/write only)
- **No Client Secret**: Designed for public OAuth clients (no client secret needed)

## Development

### Compile the Project
```bash
mix compile
```

### Run Tests
```bash
mix test
```

### Build Executable
```bash
mix escript.build
```

### Run Locally Without Building
```bash
mix run -e 'XeroCLI.CLI.main(["--help"])'
```

## Troubleshooting

### Authentication Issues

**Problem**: "No authorization code found"
- Ensure you're completing the authorization in the browser
- Check that the redirect URI is exactly `http://localhost:8888/callback`
- Make sure port 8888 is not in use by another application

**Problem**: "Token exchange failed"
- Verify your Client ID is correct
- Ensure your Xero app has the correct redirect URI configured
- Check that all required scopes are enabled in your Xero app

### API Request Issues

**Problem**: "Authentication failed. Please run 'xero auth login' again"
- Your refresh token may have expired (90 days)
- Run `xero auth login` to re-authenticate

**Problem**: "Failed to fetch organizations: No Xero organizations found"
- Ensure your Xero account has at least one organization
- Check that your Xero app has the necessary permissions

### SSL/Certificate Issues

If you encounter SSL certificate errors:
```bash
# Linux
sudo update-ca-certificates

# macOS
# Certificates are managed by Keychain Access
```

## Roadmap

Future features planned:
- [ ] Create and update invoices
- [ ] Contact management
- [ ] Bank transaction reconciliation
- [ ] Report generation
- [ ] Batch operations
- [ ] Export to CSV/JSON
- [ ] Multiple organization support
- [ ] Custom date range filtering

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

MIT License - See LICENSE file for details

## Links

- [Xero API Documentation](https://developer.xero.com/documentation/)
- [Xero Developer Portal](https://developer.xero.com/)
- [OAuth 2.0 RFC](https://tools.ietf.org/html/rfc6749)
- [PKCE RFC](https://tools.ietf.org/html/rfc7636)

## Support

For issues and questions:
- Create an issue on GitHub
- Check [Xero API Documentation](https://developer.xero.com/documentation/)
- Visit [Xero Developer Community](https://developer.xero.com/community/)
