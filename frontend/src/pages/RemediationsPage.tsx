import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { ArrowLeft, Shield, GitPullRequest, CheckCircle, XCircle, Clock } from 'lucide-react'
import { api } from '../api'
import type { Remediation } from '../types'

function StatusBadge({ status }: { status: Remediation['Status'] }) {
  const config: Record<string, { label: string; color: string }> = {
    pending: { label: 'Pending', color: 'var(--muted)' },
    approved: { label: 'Approved', color: 'var(--medium)' },
    declined: { label: 'Declined', color: 'var(--critical)' },
    pr_created: { label: 'PR Created', color: 'var(--safe)' },
    pr_merged: { label: 'Merged', color: 'var(--safe)' },
    failed: { label: 'Failed', color: 'var(--critical)' },
  }
  const { label, color } = config[status] ?? config.pending
  return (
    <span
      className="px-2 py-0.5 rounded text-xs font-medium"
      style={{ color, background: color + '18', border: '1px solid ' + color + '33' }}
    >
      {label}
    </span>
  )
}

export default function RemediationsPage() {
  const navigate = useNavigate()
  const { data: remediations, isLoading } = useQuery({
    queryKey: ['remediations'],
    queryFn: api.getRemediations,
  })

  return (
    <div className="min-h-screen" style={{ background: 'var(--background)' }}>
      <nav className="flex items-center gap-3 px-8 py-4 border-b" style={{ borderColor: 'var(--border)' }}>
        <button onClick={() => navigate('/dashboard')} style={{ color: 'var(--muted)' }}>
          <ArrowLeft size={18} />
        </button>
        <Shield size={18} style={{ color: 'var(--text)' }} />
        <span className="text-sm font-semibold" style={{ color: 'var(--text)' }}>Remediations</span>
      </nav>

      <main className="px-8 py-8 max-w-4xl mx-auto">
        <div className="mb-6">
          <h1 className="text-2xl font-semibold mb-1" style={{ color: 'var(--text)' }}>Remediations</h1>
          <p className="text-sm" style={{ color: 'var(--muted)' }}>Full audit trail of every AI-proposed fix</p>
        </div>

        {isLoading && (
          <div className="flex items-center justify-center py-20">
            <p className="text-sm" style={{ color: 'var(--muted)' }}>Loading...</p>
          </div>
        )}

        {remediations && remediations.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20">
            <Shield size={32} className="mb-4" style={{ color: 'var(--muted)' }} />
            <p className="text-sm" style={{ color: 'var(--muted)' }}>No remediations yet.</p>
          </div>
        )}

        {remediations && remediations.length > 0 && (
          <div className="space-y-3">
            {remediations.map((rem) => (
              <div key={rem.ID} className="p-5 rounded-lg" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
                <div className="flex items-start justify-between gap-4 mb-3">
                  <p className="text-sm font-medium" style={{ color: 'var(--text)' }}>{rem.What}</p>
                  <StatusBadge status={rem.Status} />
                </div>
                <p className="text-xs mb-3" style={{ color: 'var(--muted)' }}>{rem.Risk}</p>
                {rem.PRUrl && (
                  <a href={rem.PRUrl} target="_blank" rel="noopener noreferrer" className="flex items-center gap-1.5 text-xs" style={{ color: 'var(--safe)' }}>
                    View pull request #{rem.PRNumber}
                  </a>
                )}
                <div className="flex items-center gap-4 mt-3 pt-3" style={{ borderTop: '1px solid var(--border)' }}>
                  <span className="text-xs" style={{ color: 'var(--muted)' }}>
                    {new Date(rem.CreatedAt).toLocaleDateString()}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </main>
    </div>
  )
}