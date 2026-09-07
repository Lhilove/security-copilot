package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
)

const maxWebhookBodyBytes = 25 * 1024 * 1024 // 25MB - GitHub's max payload size

// VerifyWebhookSignature validates the HMAC-SHA256 signature on a GitHub webhook payload.
// The raw body must be read before calling this function.
// Verification happens before any payload parsing to ensure untrusted content
// is never processed without a valid signature.
func VerifyWebhookSignature(body []byte, signature string, secret string) error {
	if signature == "" {
		return fmt.Errorf("missing signature header")
	}

	// GitHub sends the signature as "sha256=<hex>"
	const prefix = "sha256="
	if len(signature) <= len(prefix) {
		return fmt.Errorf("malformed signature header")
	}

	sigHex := signature[len(prefix):]
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return fmt.Errorf("invalid signature encoding")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := mac.Sum(nil)

	// Constant-time comparison to prevent timing attacks
	if !hmac.Equal(sigBytes, expected) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// ReadWebhookBody reads and size-limits the webhook request body.
func ReadWebhookBody(r *http.Request) ([]byte, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxWebhookBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("read webhook body: %w", err)
	}
	return body, nil
}
