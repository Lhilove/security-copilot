# Security Copilot

AI-powered security remediation for GitHub repositories. Connects to your repos, explains vulnerabilities in plain English, and opens pull requests with the fix. You just review and approve.

Live at [securitycopilot.com.ng](https://securitycopilot.com.ng) — or self-host in minutes with Docker.

---

## What it does

Most security tools tell you something is broken. Security Copilot tells you why it matters and fixes it.

Connect a GitHub repository. Security Copilot pulls in findings from CodeQL, Dependabot, and Secret Scanning. For each finding, the AI explains the business impact in plain English, proposes a fix, and — with your approval — opens a pull request. For Dependabot alerts, it bumps the vulnerable dependency to the patched version automatically.

If CodeQL and Dependabot are not enabled on a repository, Security Copilot enables them automatically and runs its own AI-powered scan of the codebase immediately, so you get findings right away rather than waiting for the next CI run.

```
Repository connected
        |
        v
Security features enabled automatically
(CodeQL workflow committed, Dependabot alerts on, Secret Scanning on)
        |
        v
AI scans codebase directly for immediate findings
        |
        v
Findings aggregated from all sources
        |
        v
Click a finding → AI explains it in plain English
        |
        v
Approve → pull request created automatically
        |
        v
Fix ships
```

---

## Features

- GitHub OAuth — connect repositories with one click
- Aggregates CodeQL, Dependabot, and Secret Scanning findings
- Auto-enables security features on connected repositories
- AI direct scan of the codebase using DeepSeek Coder via Ollama
- Plain-English explanation of every finding (what it is, why it matters, how to fix it)
- Proposed code fix with one-click pull request creation
- Automatic dependency version bumping for Dependabot alerts
- Notification channels: email (SendByte), Slack, Discord, Telegram
- Signed approval tokens — approve or decline fixes directly from any notification
- Risk scoring per repository
- AES-256-GCM encryption for all stored GitHub tokens

---

## Self-hosting

The fastest way to run Security Copilot locally is Docker Compose. It starts PostgreSQL, Ollama, and pulls the DeepSeek Coder model automatically.

### Prerequisites

- Docker and Docker Compose
- A GitHub OAuth app ([create one here](https://github.com/settings/applications/new))
  - Set the callback URL to `http://localhost:8080/api/v1/auth/github/callback`

### Setup

**1. Clone the repository**

```bash
git clone https://github.com/lhilove/security-copilot.git
cd security-copilot
```

**2. Build the frontend**

```bash
cd frontend
npm install
npm run build
cd ..
```

**3. Configure environment variables**

```bash
cp backend/.env.example backend/.env
```

Generate the required secret keys:

```bash
# Linux / macOS
bash scripts/generate-keys.sh

# Windows (PowerShell)
.\scripts\generate-keys.ps1
```

Copy the output into `backend/.env` and fill in your GitHub OAuth credentials:

```env
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GITHUB_REDIRECT_URL=http://localhost:8080/api/v1/auth/github/callback
GITHUB_WEBHOOK_SECRET=any_random_string
ENCRYPTION_KEY=generated_above
JWT_SECRET=generated_above
AI_PROVIDER=ollama
OLLAMA_BASE_URL=http://ollama:11434
AI_MODEL=deepseek-coder:6.7b
FRONTEND_URL=http://localhost:8080
BASE_URL=http://localhost:8080
```

**4. Start everything**

```bash
docker compose up -d
```

This starts PostgreSQL and Ollama, then pulls the DeepSeek Coder model (3.8 GB — takes a few minutes on first run). The API starts automatically once both are ready.

**5. Open the app**

Navigate to `http://localhost:8080` and connect your GitHub account.

> The model download happens once. Subsequent starts are fast.

---

## Architecture

```
React Frontend
      |
      v
Go REST API (Gin)
      |
      +-- PostgreSQL (findings, remediations, users)
      |
      +-- GitHub API (OAuth, findings, pull requests)
      |
      +-- Ollama (DeepSeek Coder 6.7B — local AI, no external API calls)
      |
      +-- SendByte / Slack / Discord / Telegram (notifications)
```

The AI runs entirely on your own infrastructure. No code or findings are sent to OpenAI, Anthropic, or any external AI service.

---

## Security design

Security controls are applied to the product itself, not only the code it analyzes.

- GitHub access tokens are encrypted at rest with AES-256-GCM
- OAuth state is validated on every callback using constant-time comparison
- State cookies are HttpOnly and deleted immediately after validation
- Approval tokens for notification channels are signed JWTs with 24-hour expiry
- Prompt injection defenses are applied at the Go layer before any content reaches the AI model
- Code snippets sent for AI analysis are labeled as untrusted data in the system prompt
- Encryption keys are never committed to version control
- All AI analysis is sandboxed — the model cannot execute code or make network requests

---

## Tech stack

| Layer | Technology |
|---|---|
| Backend | Go, Gin |
| Database | PostgreSQL, golang-migrate |
| AI | DeepSeek Coder 6.7B via Ollama |
| Frontend | React, TypeScript, Vite, TanStack Query |
| Auth | GitHub OAuth 2.0, JWT |
| Encryption | AES-256-GCM |
| Infrastructure | Oracle Cloud (ARM A1 Flex), Nginx, Let's Encrypt |
| Notifications | SendByte, Slack, Discord, Telegram |

---

## Project structure

```
security-copilot/
├── backend/
│   ├── cmd/api/           # Entry point, server setup, HTTP handlers
│   ├── internal/
│   │   ├── ai/            # AI provider abstraction (Ollama, Nvidia)
│   │   ├── auth/          # GitHub OAuth, JWT
│   │   ├── config/        # Environment configuration
│   │   ├── crypto/        # AES-256-GCM encryption
│   │   ├── database/      # PostgreSQL repositories
│   │   ├── findings/      # Finding normalization and sync
│   │   ├── github/        # GitHub API client
│   │   ├── notifications/ # Email, Slack, Discord, Telegram dispatch
│   │   ├── repositories/  # Repository management
│   │   ├── scanner/       # AI direct codebase scanner
│   │   └── securitysetup/ # Auto-enable GitHub security features
│   ├── migrations/        # SQL up/down migrations
│   ├── Dockerfile
│   └── .env.example
├── frontend/
│   └── src/
│       ├── api/           # API client
│       ├── components/    # NotificationBell
│       └── pages/         # Dashboard, Repository, Finding, Settings
├── scripts/
│   ├── generate-keys.sh
│   └── generate-keys.ps1
├── docker-compose.yml
└── README.md
```

---

## Running tests

```bash
cd backend
go test ./...
```

---

## Privacy

Security Copilot does not send your code to any external AI service. The AI model runs on your own infrastructure (or ours, for the hosted version). Read the full [Privacy Policy](https://securitycopilot.com.ng/privacy.html).

---

## License

MIT — [github.com/Lhilove/security-copilot](https://github.com/Lhilove/security-copilot)