package notifications

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ApprovalAction is the action encoded in the approval token.
type ApprovalAction string

const (
	ActionApprove ApprovalAction = "approve"
	ActionDecline ApprovalAction = "decline"
)

// ApprovalClaims is the JWT payload for channel-based approval links.
type ApprovalClaims struct {
	UserID    string         `json:"user_id"`
	FindingID string         `json:"finding_id"`
	Action    ApprovalAction `json:"action"`
	Channel   string         `json:"channel"` // email, slack, discord, telegram
	jwt.RegisteredClaims
}

// IssueApprovalToken creates a signed approval token valid for 24 hours.
func IssueApprovalToken(userID, findingID string, action ApprovalAction, channel string, secret []byte) (string, error) {
	claims := ApprovalClaims{
		UserID:    userID,
		FindingID: findingID,
		Action:    action,
		Channel:   channel,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("sign approval token: %w", err)
	}

	return signed, nil
}

// ValidateApprovalToken parses and validates an approval token.
func ValidateApprovalToken(tokenString string, secret []byte) (*ApprovalClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &ApprovalClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse approval token: %w", err)
	}

	claims, ok := token.Claims.(*ApprovalClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid approval token claims")
	}

	return claims, nil
}
