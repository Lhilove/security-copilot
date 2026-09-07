import { useQuery, useMutation } from '@tanstack/react-query'
import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, RefreshCw, AlertTriangle, Shield } from 'lucide-react'
import { api } from '../api'
import type { Finding } from '../types'

// Severity badge component
function SeverityBadge({ severity }: { severity: Finding['Severity'] }) {
  const colors: Record<string, string> = {
    critical: 'var(--critical)',
    high: 'var(--high)',
    medium: 'var(--medium)',
    low: 'var(--low)',
  }

  return (
    <span
      className="px-2 py-0.5 rounded text-xs font-medium"
      style={{
        color: colors[severity],
        background: `${colors[severity]}18`,
        border: `1px solid ${colors[severity]}33`,
      }}
    >
      {severity}
    </span>
  )
}

// Source label
function SourceBadge({ source }: { source: Finding['Source'] }) {
  const labels: Record<string, string> = {
    code_scanning: 'CodeQL',
    dependabot: 'Dependabot',
    secret_scanning: 'Secret',
  }

  return (
    <span className="text-xs px-2 py-0.5 rounded" style={{ background: 'var(--surface-2)', color: 'var(--muted)' }}>
      {labels[source] ?? source}
    </span>
  )
}

export default function RepositoryPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  // Fetch findings for this repository
  const { data: findings, isLoading, error } = useQuery({
    queryKey: ['findings', id],
    queryFn: () => api.getFindings(id!),
    enabled: !!id,
  })

  // Fetch overview for risk score
  const { data: overview } = useQuery({
    queryKey: ['overview', id],
    queryFn: () => api.getRepositoryOverview(id!),
    enabled: !!id,
  })

  // Sync findings mutation
  const sync = useMutation({
    mutationFn: () => api.syncFindings(id!),
  })

  const critical = findings?.filter(f => f.Severity === 'critical') ?? []
  const high = findings?.filter(f => f.Severity === 'high') ?? []
  const medium = findings?.filter(f => f.Severity === 'medium') ?? []
  const low = findings?.filter(f => f.Severity === 'low') ?? []

  return (
    <div className="min-h-screen" style={{ background: 'var(--background)' }}>

      {/* Nav */}
      <nav className="flex items-center justify-between px-8 py-4 border-b" style={{ borderColor: 'var(--border)' }}>
        <div className="flex items-center gap-3">
          <button onClick={() => navigate('/dashboard')} style={{ color: 'var(--muted)' }}>
            <ArrowLeft size={18} />
          </button>
          <Shield size={18} style={{ color: 'var(--text)' }} />
          <span className="text-sm font-semibold" style={{ color: 'var(--text)' }}>
            {overview?.repository ?? 'Repository'}
          </span>
        </div>
        <button
          onClick={() => sync.mutate()}
          disabled={sync.isPending}
          className="flex items-center gap-1.5 text-xs transition-all"
          style={{ color: 'var(--muted)' }}
        >
          <RefreshCw size={12} className={sync.isPending ? 'animate-spin' : ''} />
          {sync.isPending ? 'Syncing...' : 'Sync'}
        </button>
      </nav>

      <main className="px-8 py-8 max-w-5xl mx-auto">

        {/* Risk overview */}
        {overview && (
          <div className="grid grid-cols-5 gap-3 mb-8">
            {[
              { label: 'Risk Score', value: overview.security.risk_score, suffix: '/100' },
              { label: 'Critical', value: overview.security.findings.critical, color: 'var(--critical)' },
              { label: 'High', value: overview.security.findings.high, color: 'var(--high)' },
              { label: 'Medium', value: overview.security.findings.medium, color: 'var(--medium)' },
              { label: 'Low', value: overview.security.findings.low, color: 'var(--low)' },
            ].map((stat) => (
              <div
                key={stat.label}
                className="p-4 rounded-lg"
                style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}
              >
                <p className="text-xs mb-1" style={{ color: 'var(--muted)' }}>{stat.label}</p>
                <p className="text-2xl font-semibold" style={{ color: stat.color ?? 'var(--text)' }}>
                  {stat.value}{stat.suffix ?? ''}
                </p>
              </div>
            ))}
          </div>
        )}

        {/* Loading */}
        {isLoading && (
          <div className="flex items-center justify-center py-20">
            <p className="text-sm" style={{ color: 'var(--muted)' }}>Loading findings...</p>
          </div>
        )}

        {/* Error */}
        {error && (
          <div className="p-4 rounded-lg mb-6" style={{ background: 'rgba(255,68,68,0.1)', border: '1px solid rgba(255,68,68,0.2)' }}>
            <p className="text-sm" style={{ color: 'var(--critical)' }}>Failed to load findings.</p>
          </div>
        )}

        {/* Findings list */}
        {findings && findings.length > 0 && (
          <div>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-sm font-medium" style={{ color: 'var(--text)' }}>
                {findings.length} findings
              </h2>
            </div>

            <div className="space-y-2">
              {findings.map((finding) => (
                <div
                  key={finding.ID}
                  className="flex items-center gap-4 p-4 rounded-lg cursor-pointer transition-all hover:border-white/10"
                  style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}
                  onClick={() => navigate(`/findings/${finding.ID}`)}
                >
                  <AlertTriangle size={14} style={{ color: 'var(--muted)', flexShrink: 0 }} />

                  <div className="flex-1 min-w-0">
                    <p className="text-sm truncate" style={{ color: 'var(--text)' }}>
                      {finding.Title}
                    </p>
                    {finding.FilePath && (
                      <p className="text-xs mt-0.5 font-mono truncate" style={{ color: 'var(--muted)' }}>
                        {finding.FilePath}{finding.LineNumber ? `:${finding.LineNumber}` : ''}
                      </p>
                    )}
                    {finding.PackageName && (
                      <p className="text-xs mt-0.5" style={{ color: 'var(--muted)' }}>
                        {finding.PackageName} {finding.CVEID && `· ${finding.CVEID}`}
                      </p>
                    )}
                  </div>

                  <div className="flex items-center gap-2 shrink-0">
                    <SourceBadge source={finding.Source} />
                    <SeverityBadge severity={finding.Severity} />
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Empty state */}
        {findings && findings.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20">
            <Shield size={32} className="mb-4" style={{ color: 'var(--muted)' }} />
            <p className="text-sm" style={{ color: 'var(--muted)' }}>
              No findings yet. Click Sync to fetch from GitHub.
            </p>
          </div>
        )}
      </main>
    </div>
  )
}