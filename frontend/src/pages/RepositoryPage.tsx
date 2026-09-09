import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, RefreshCw, AlertTriangle, Shield, ShieldCheck, Bot } from 'lucide-react'
import { api } from '../api'
import type { Finding } from '../types'
import NotificationBell from '../components/NotificationBell'

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

function SourceBadge({ source }: { source: Finding['Source'] }) {
  const config: Record<string, { label: string; color: string; bg: string; icon?: React.ReactNode }> = {
    code_scanning: {
      label: 'CodeQL',
      color: 'var(--muted)',
      bg: 'var(--surface-2)',
    },
    dependabot: {
      label: 'Dependabot',
      color: 'var(--muted)',
      bg: 'var(--surface-2)',
    },
    secret_scanning: {
      label: 'Secret',
      color: 'var(--muted)',
      bg: 'var(--surface-2)',
    },
    ai_scan: {
      label: 'AI Scan',
      color: '#a78bfa',
      bg: 'rgba(167,139,250,0.1)',
      icon: <Bot size={10} />,
    },
  }

  const c = config[source] ?? { label: source, color: 'var(--muted)', bg: 'var(--surface-2)' }

  return (
    <span
      className="flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium"
      style={{ color: c.color, background: c.bg, border: `1px solid ${c.color}33` }}
    >
      {c.icon}
      {c.label}
    </span>
  )
}

function ScanBanner() {
  return (
    <div
      className="flex items-center gap-3 p-4 rounded-lg mb-6"
      style={{
        background: 'rgba(167,139,250,0.08)',
        border: '1px solid rgba(167,139,250,0.2)',
      }}
    >
      <Bot size={16} style={{ color: '#a78bfa', flexShrink: 0 }} />
      <div>
        <p className="text-sm font-medium" style={{ color: '#a78bfa' }}>
          AI scan in progress
        </p>
        <p className="text-xs mt-0.5" style={{ color: 'var(--muted)' }}>
          DeepSeek is scanning your codebase. New findings will appear automatically.
        </p>
      </div>
    </div>
  )
}

export default function RepositoryPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const { data: findings, isLoading, error } = useQuery({
    queryKey: ['findings', id],
    queryFn: () => api.getFindings(id!),
    enabled: !!id,
    refetchInterval: 30000, // poll every 30s to pick up AI scan findings
  })

  const { data: overview } = useQuery({
    queryKey: ['overview', id],
    queryFn: () => api.getRepositoryOverview(id!),
    enabled: !!id,
    refetchInterval: 30000,
  })

  const sync = useMutation({
    mutationFn: () => api.syncFindings(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['findings', id] })
      queryClient.invalidateQueries({ queryKey: ['overview', id] })
    },
  })

  const setupSecurity = useMutation({
    mutationFn: () => api.setupSecurity(id!),
  })

  const hasAIScanFindings = findings?.some(f => f.Source === 'ai_scan')
  const aiScanRunning = setupSecurity.isPending

  return (
    <div className="min-h-screen" style={{ background: 'var(--background)' }}>

      {/* Nav */}
      <nav
        className="flex items-center justify-between px-8 py-4 border-b"
        style={{ borderColor: 'var(--border)' }}
      >
        <div className="flex items-center gap-3">
          <button onClick={() => navigate('/dashboard')} style={{ color: 'var(--muted)' }}>
            <ArrowLeft size={18} />
          </button>
          <Shield size={18} style={{ color: 'var(--text)' }} />
          <span className="text-sm font-semibold" style={{ color: 'var(--text)' }}>
            {overview?.repository ?? 'Repository'}
          </span>
        </div>
        <div className="flex items-center gap-3">
          {/* Enable Security button */}
          <button
            onClick={() => setupSecurity.mutate()}
            disabled={setupSecurity.isPending || setupSecurity.isSuccess}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all"
            style={{
              background: setupSecurity.isSuccess
                ? 'rgba(22,163,74,0.1)'
                : 'rgba(167,139,250,0.1)',
              color: setupSecurity.isSuccess ? '#16a34a' : '#a78bfa',
              border: `1px solid ${setupSecurity.isSuccess
                ? 'rgba(22,163,74,0.2)'
                : 'rgba(167,139,250,0.2)'}`,
            }}
          >
            <ShieldCheck size={12} />
            {setupSecurity.isPending
              ? 'Starting scan...'
              : setupSecurity.isSuccess
              ? 'Scan started'
              : 'Enable Security & Scan'}
          </button>

          {/* Sync button */}
          <button
            onClick={() => sync.mutate()}
            disabled={sync.isPending}
            className="flex items-center gap-1.5 text-xs transition-all"
            style={{ color: 'var(--muted)' }}
          >
            <RefreshCw size={12} className={sync.isPending ? 'animate-spin' : ''} />
            {sync.isPending ? 'Syncing...' : 'Sync'}
          </button>

          <NotificationBell />
        </div>
      </nav>

      <main className="px-8 py-8 max-w-5xl mx-auto">

        {/* AI scan in progress banner */}
        {aiScanRunning && <ScanBanner />}

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

        {/* AI scan findings info bar */}
        {hasAIScanFindings && (
          <div
            className="flex items-center gap-2 px-4 py-2.5 rounded-lg mb-4"
            style={{
              background: 'rgba(167,139,250,0.05)',
              border: '1px solid rgba(167,139,250,0.15)',
            }}
          >
            <Bot size={13} style={{ color: '#a78bfa' }} />
            <p className="text-xs" style={{ color: 'var(--muted)' }}>
              Some findings were detected by AI direct scan, marked with the{' '}
              <span style={{ color: '#a78bfa' }}>AI Scan</span> badge.
            </p>
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
          <div
            className="p-4 rounded-lg mb-6"
            style={{ background: 'rgba(255,68,68,0.1)', border: '1px solid rgba(255,68,68,0.2)' }}
          >
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
            <p className="text-sm mb-2" style={{ color: 'var(--muted)' }}>
              No findings yet.
            </p>
            <p className="text-xs" style={{ color: 'var(--muted)' }}>
              Click "Enable Security & Scan" to run an AI scan or sync from GitHub.
            </p>
          </div>
        )}
      </main>
    </div>
  )
}