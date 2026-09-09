package main

import (
	"github.com/lhilove/security-copilot/internal/ai"
	"github.com/lhilove/security-copilot/internal/auth"
	"github.com/lhilove/security-copilot/internal/config"
	"github.com/lhilove/security-copilot/internal/database"
	"github.com/lhilove/security-copilot/internal/findings"
	gh "github.com/lhilove/security-copilot/internal/github"
	"github.com/lhilove/security-copilot/internal/notifications"
	"github.com/lhilove/security-copilot/internal/repositories"
	"github.com/lhilove/security-copilot/internal/securitysetup"
)

// Server holds all dependencies and is the receiver for all HTTP handlers.
type Server struct {
	cfg               *config.Config
	authService       *auth.Service
	repoService       *repositories.Service
	findingService    *findings.Service
	webhookSvc        *gh.WebhookHandler
	repoRepo          *database.RepositoryRepository
	findingRepo       *database.FindingRepository
	remediationRepo   *database.RemediationRepository
	aiProvider        ai.Provider
	securitySetup     *securitysetup.Service
	notifDispatcher   *notifications.Dispatcher
	notifSettingsRepo *database.SettingsRepository
	notifRepo         *database.NotificationRepository
}
