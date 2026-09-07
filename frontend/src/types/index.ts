export interface Repository {
  ID: string
  UserID: string
  GitHubRepoID: number
  Owner: string
  Name: string
  FullName: string
  Private: boolean
  Monitored: boolean
}

export interface Finding {
  ID: string
  RepositoryID: string
  Source: 'code_scanning' | 'dependabot' | 'secret_scanning'
  SourceAlertID: string
  Severity: 'critical' | 'high' | 'medium' | 'low'
  Title: string
  Description: string
  State: string
  FilePath: string
  LineNumber: number | null
  PackageName: string
  CVEID: string
  SecretType: string
}

export interface Remediation {
  ID: string
  FindingID: string
  UserID: string
  What: string
  Risk: string
  Fix: string
  ProposedCode: string
  CanAutoFix: boolean
  Status: 'pending' | 'approved' | 'declined' | 'pr_created' | 'pr_merged' | 'failed'
  PRUrl: string
  PRNumber: number | null
  ApprovedAt: string | null
  DeclinedAt: string | null
  CreatedAt: string
  UpdatedAt: string
}

export interface AnalysisResult {
  remediation_id: string
  finding_id: string
  what: string
  risk: string
  fix: string
  proposed_code: string
  can_auto_fix: boolean
  status: string
}

export interface SecurityOverview {
  repository: string
  monitored: boolean
  security: {
    risk_score: number
    findings: {
      critical: number
      high: number
      medium: number
      low: number
      total: number
    }
  }
}
