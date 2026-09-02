package github

// WebhookRepository contains only the fields we need from the repository
// object in a webhook payload. We use the node_id to look up our own
// database record — we never use URLs from the payload for outbound requests.
type WebhookRepository struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
}

// CodeScanningAlertEvent represents a code_scanning_alert webhook event.
type CodeScanningAlertEvent struct {
	Action string `json:"action"` // created, fixed, dismissed, reopened
	Alert  struct {
		Number int    `json:"number"`
		State  string `json:"state"`
		Rule   struct {
			ID          string `json:"id"`
			Description string `json:"description"`
			Severity    string `json:"severity"`
		} `json:"rule"`
		MostRecentInstance struct {
			Location struct {
				Path      string `json:"path"`
				StartLine int    `json:"start_line"`
			} `json:"location"`
		} `json:"most_recent_instance"`
	} `json:"alert"`
	Repository WebhookRepository `json:"repository"`
}

// DependabotAlertEvent represents a dependabot_alert webhook event.
type DependabotAlertEvent struct {
	Action string `json:"action"` // created, dismissed, fixed, reintroduced, auto_dismissed, auto_reopened
	Alert  struct {
		Number                int    `json:"number"`
		State                 string `json:"state"`
		SecurityVulnerability struct {
			Package struct {
				Name string `json:"name"`
			} `json:"package"`
			Severity string `json:"severity"`
		} `json:"security_vulnerability"`
		SecurityAdvisory struct {
			Summary     string `json:"summary"`
			Description string `json:"description"`
			CVEId       string `json:"cve_id"`
		} `json:"security_advisory"`
	} `json:"alert"`
	Repository WebhookRepository `json:"repository"`
}

// SecretScanningAlertEvent represents a secret_scanning_alert webhook event.
type SecretScanningAlertEvent struct {
	Action string `json:"action"` // created, resolved, reopened
	Alert  struct {
		Number     int    `json:"number"`
		State      string `json:"state"`
		SecretType string `json:"secret_type"`
	} `json:"alert"`
	Repository WebhookRepository `json:"repository"`
}
