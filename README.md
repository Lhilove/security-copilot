# Security Copilot

AI-powered application security remediation for GitHub repositories.

> Built for developers, not security teams.

---

## The problem

Modern development teams already have scanners. The problem is what happens after a vulnerability is found.

- Is this actually important?
- What is the business impact?
- What should I fix first?
- How do I fix it safely?
- Can I get the fix reviewed without waiting for another team?

Security tooling answers the first question. Security Copilot answers the rest.

---

## What it does

Security Copilot connects to a developer's GitHub repositories, aggregates security findings, and guides the developer from finding to merged fix without requiring AppSec team intervention at every step.

```
Security finding
      ↓
Understand (why does this matter?)
      ↓
Prioritize (what do I fix first?)
      ↓
AI proposes remediation
      ↓
Developer reviews and approves
      ↓
Pull request created
      ↓
Security validation runs
      ↓
Fix ships
```

The AI proposes and executes controlled security work. The developer retains approval authority at every step.

---

## Status

Active development. Phase 1, 2 complete.

### Phase 1 — Foundation

- [x] Go backend
- [x] Health endpoint
- [x] GitHub OAuth
- [x] CSRF state protection (constant-time comparison)
- [x] AES-256-GCM token encryption
- [x] Encryption unit tests (correctness, nonce uniqueness, tamper detection, wrong-key rejection)
- [x] PostgreSQL with Docker Compose
- [x] Database migrations (golang-migrate)
- [x] User persistence
- [x] Encrypted GitHub token storage

### Phase 2 — GitHub integration

- [x] Repository discovery
- [x] Repository persistence
- [x] Repository selection
- [x] GitHub API service abstraction
- [x] Webhook verification
- [x] Repository event ingestion

### Phase 3 — Security intelligence

- [x] Security finding model
- [x] Finding normalization and aggregation
- [x] Severity normalization
- [x] Risk scoring
- [x] Business impact analysis

### Phase 4 — AI remediation

- [x] AI provider abstraction
- [x] Vulnerability explanation
- [x] Remediation generation
- [x] Output validation
- [x] Prompt injection defenses

### Phase 5 — Controlled remediation

- [x] Remediation approval workflow
- [x] Pull request generation
- [ ] Security validation
- [x] Audit logging

### Phase 6 — Notifications

- [ ] Slack, Discord, Telegram integration
- [ ] Secure approval workflow tied to server-side verification

### Phase 7 — Frontend

- [ ] React dashboard
- [ ] Repository security overview
- [ ] Finding details and business impact view
- [ ] Remediation review and approval interface

---

## Architecture

```
Developer
    ↓
React Dashboard (planned)
    ↓
Go REST API
    ↓
PostgreSQL
    ↓
GitHub API
    ↓
AI Provider (planned)
```

The backend is written in Go. PostgreSQL is the persistent datastore. The AI layer is provider-agnostic by design.

---

## Security design

Security controls are applied to the product itself, not only the code it analyzes.

- GitHub access tokens are encrypted at rest with AES-256-GCM before database storage
- OAuth state is validated on every callback using constant-time comparison to prevent login CSRF
- State cookies are HttpOnly and deleted immediately after validation
- Encryption keys are never committed to version control
- Database migrations run with up/down support
- The AI remediation layer will include output validation and prompt injection defenses before any code change is proposed

---

## Getting started

### Prerequisites

- Go 1.22+
- Docker and Docker Compose
- A GitHub OAuth application ([create one here](https://github.com/settings/applications/new))

### Setup

1. Clone the repository

```bash
git clone https://github.com/lhilove/security-copilot.git
cd security-copilot
```

2. Start PostgreSQL

```bash
docker compose up -d
```

3. Configure environment variables

```bash
cd backend
cp .env.example .env
```

Edit `.env`:

```env
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GITHUB_REDIRECT_URL=http://localhost:8080/api/v1/auth/github/callback
DATABASE_URL=postgres://security_copilot:security_copilot_dev@localhost:5432/security_copilot
ENCRYPTION_KEY=your_32_byte_base64_encoded_key
MIGRATIONS_PATH=migrations
```

Generate an encryption key:

```bash
# PowerShell
$bytes = New-Object byte[] 32
[System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
[Convert]::ToBase64String($bytes)

# Linux / macOS
openssl rand -base64 32
```

4. Run the API

```bash
go run ./cmd/api
```

Migrations run automatically on startup.

5. Test the health endpoint

```bash
curl http://localhost:8080/health
```

```json
{"status":"ok","service":"security-copilot-api"}
```

### GitHub OAuth flow

Navigate to `http://localhost:8080/api/v1/auth/github` to begin the OAuth flow. After authorization, your GitHub identity and encrypted access token are persisted to the database.

---

## Project structure

```
Security-Copilot/
├── backend/
│   ├── cmd/
│   │   └── api/
│   │       └── main.go
│   ├── internal/
│   │   ├── auth/          # GitHub OAuth
│   │   ├── config/        # Environment configuration
│   │   ├── crypto/        # AES-256-GCM encryption
│   │   └── database/      # PostgreSQL connection, migrations, repositories
│   └── migrations/        # SQL up/down migrations
├── docker-compose.yml
└── .gitignore
```

---

## Running tests

```bash
cd backend
go test ./...
```

Current test coverage: `internal/crypto` (encryption/decryption, nonce uniqueness, tamper detection, wrong-key rejection).

---

## What success looks like

The project is not measured by number of features or AI calls. The meaningful metrics are:

- **Time to remediation** — how long from finding to merged fix
- **Developer comprehension** — can the developer understand why the issue matters
- **Remediation quality** — does the fix address the vulnerability without introducing a new one
- **Developer autonomy** — can developers resolve security issues without waiting for AppSec intervention

---

## License

MIT