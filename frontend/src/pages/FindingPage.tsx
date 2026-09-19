import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, Shield, CheckCircle, XCircle, GitPullRequest, Loader, AlertTriangle } from 'lucide-react'
import { api } from '../api'
import type { AnalysisResult } from '../types'
import NotificationBell from '../components/NotificationBell'

export default function FindingPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [analysis, setAnalysis] = useState<AnalysisResult | null>(null)
  const [prUrl, setPrUrl] = useState<string | null>(null)
  const [reviewed, setReviewed] = useState(false)

  const analyze = useMutation({
    mutationFn: () => api.analyzeFinding(id!),
    onSuccess: (result) => setAnalysis(result),
  })

  const createPR = useMutation({
    mutationFn: () => api.createPR(id!),
    onSuccess: (result) => setPrUrl(result.pr_url),
  })

  const approve = useMutation({
    mutationFn: () => api.approveFinding(id!),
    onSuccess: (result) => {
      // Backend now tells us whether a PR can be created
      if (result.can_auto_fix) {
        setTimeout(() => createPR.mutate(), 500)
      } else {
        setReviewed(true)
      }
    },
  })

  const decline = useMutation({
    mutationFn: () => api.declineFinding(id!),
    onSuccess: () => navigate(-1),
  })

  const isApproving = approve.isPending || createPR.isPending

  // canAutoFix from analysis state — used to show correct button before approving
  const canAutoFix = !!(analysis?.proposed_code && analysis.proposed_code.trim() !== '')

  return (
    <div className="min-h-screen" style={{ background: 'var(--background)' }}>
      <nav className="flex items-center justify-between px-8 py-4 border-b" style={{ borderColor: 'var(--border)' }}>
        <div className="flex items-center gap-3">
          <button onClick={() => navigate(-1)} style={{ color: 'var(--muted)' }}>
            <ArrowLeft size={18} />
          </button>
          <Shield size={18} style={{ color: 'var(--text)' }} />
          <span className="text-sm font-semibold" style={{ color: 'var(--text)' }}>Finding Analysis</span>
        </div>
        <NotificationBell />
      </nav>

      <main className="px-8 py-8 max-w-3xl mx-auto">

        {/* PR created success banner */}
        {prUrl && (
          <div className="p-5 rounded-lg mb-6 flex items-start gap-3" style={{ background: 'rgba(34,197,94,0.1)', border: '1px solid rgba(34,197,94,0.2)' }}>
            <GitPullRequest size={18} style={{ color: 'var(--safe)', flexShrink: 0, marginTop: 1 }} />
            <div>
              <p className="text-sm font-medium mb-1" style={{ color: 'var(--safe)' }}>Pull request created</p>
              <a href={prUrl} target="_blank" rel="noopener noreferrer" className="text-xs underline" style={{ color: 'var(--safe)' }}>
                {prUrl}
              </a>
            </div>
          </div>
        )}

        {/* Reviewed success banner — shown when no auto fix available */}
        {reviewed && !prUrl && (
          <div className="p-5 rounded-lg mb-6 flex items-start gap-3" style={{ background: 'rgba(34,197,94,0.1)', border: '1px solid rgba(34,197,94,0.2)' }}>
            <CheckCircle size={18} style={{ color: 'var(--safe)', flexShrink: 0, marginTop: 1 }} />
            <div>
              <p className="text-sm font-medium" style={{ color: 'var(--safe)' }}>Marked as reviewed</p>
              <p className="text-xs mt-1" style={{ color: 'var(--muted)' }}>Apply the fix manually using the guidance above.</p>
            </div>
          </div>
        )}

        {/* Initial state — not yet analyzed */}
        {!analysis && !analyze.isPending && (
          <div className="p-8 rounded-lg text-center" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
            <Shield size={32} className="mx-auto mb-4" style={{ color: 'var(--muted)' }} />
            <p className="text-sm mb-2" style={{ color: 'var(--text)' }}>Analyze this finding with AI</p>
            <p className="text-xs mb-6" style={{ color: 'var(--muted)' }}>AI will explain the business impact and propose a fix</p>
            {analyze.isError && (
              <p className="text-xs mb-4" style={{ color: 'var(--critical)' }}>Analysis failed. Try again.</p>
            )}
            <button
              onClick={() => analyze.mutate()}
              className="px-5 py-2.5 rounded-lg text-sm font-medium"
              style={{ background: 'var(--accent)', color: '#000' }}
            >
              Analyze finding
            </button>
          </div>
        )}

        {/* Loading state */}
        {analyze.isPending && (
          <div className="p-8 rounded-lg text-center" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
            <Loader size={24} className="mx-auto mb-4 animate-spin" style={{ color: 'var(--muted)' }} />
            <p className="text-sm" style={{ color: 'var(--muted)' }}>Analyzing with AI...</p>
            <p className="text-xs mt-2" style={{ color: 'var(--muted)' }}>This may take a minute</p>
          </div>
        )}

        {/* Analysis result */}
        {analysis && (
          <div className="space-y-4">

            {[
              { label: 'What', value: analysis.what },
              { label: 'Risk', value: analysis.risk },
              { label: 'Fix', value: analysis.fix },
            ].map((item) => (
              <div key={item.label} className="p-5 rounded-lg" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
                <p className="text-xs font-medium uppercase tracking-wider mb-2" style={{ color: 'var(--muted)' }}>{item.label}</p>
                <p className="text-sm leading-relaxed" style={{ color: 'var(--text)' }}>{item.value}</p>
              </div>
            ))}

            {/* Proposed code — only when AI returned one */}
            {canAutoFix && (
              <div className="p-5 rounded-lg" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
                <p className="text-xs font-medium uppercase tracking-wider mb-3" style={{ color: 'var(--muted)' }}>Proposed fix</p>
                <pre className="text-xs leading-relaxed overflow-x-auto p-4 rounded" style={{ background: 'var(--surface-2)', color: 'var(--text)', fontFamily: 'monospace' }}>
                  {analysis.proposed_code}
                </pre>
              </div>
            )}

            {/* Manual fix notice */}
            {!canAutoFix && (
              <div className="p-4 rounded-lg flex items-start gap-3" style={{ background: 'rgba(255,200,0,0.08)', border: '1px solid rgba(255,200,0,0.2)' }}>
                <AlertTriangle size={15} style={{ color: '#f59e0b', flexShrink: 0, marginTop: 1 }} />
                <div>
                  <p className="text-xs font-medium mb-1" style={{ color: '#f59e0b' }}>Manual fix required</p>
                  <p className="text-xs" style={{ color: 'var(--muted)' }}>
                    The AI couldn't generate an automated patch for this finding. Follow the fix guidance above and apply it manually to your codebase.
                  </p>
                </div>
              </div>
            )}

            {/* PR creation error */}
            {createPR.isError && (
              <div className="p-4 rounded-lg" style={{ background: 'rgba(255,68,68,0.1)', border: '1px solid rgba(255,68,68,0.2)' }}>
                <p className="text-xs" style={{ color: 'var(--critical)' }}>PR creation failed. Apply the fix manually using the proposed code above.</p>
              </div>
            )}

            {/* Action buttons — hidden once reviewed or PR created */}
            {!prUrl && !reviewed && (
              <div className="flex gap-3 pt-2">
                {canAutoFix ? (
                  <button
                    onClick={() => approve.mutate()}
                    disabled={isApproving}
                    className="flex items-center justify-center gap-2 px-5 py-2.5 rounded-lg text-sm font-medium flex-1"
                    style={{ background: 'var(--safe)', color: '#000', opacity: isApproving ? 0.7 : 1 }}
                  >
                    {isApproving
                      ? <span className="flex items-center gap-2"><Loader size={14} className="animate-spin" /> Creating PR...</span>
                      : <span className="flex items-center gap-2"><CheckCircle size={14} /> Approve and create PR</span>
                    }
                  </button>
                ) : (
                  <button
                    onClick={() => approve.mutate()}
                    disabled={approve.isPending}
                    className="flex items-center justify-center gap-2 px-5 py-2.5 rounded-lg text-sm font-medium flex-1"
                    style={{ background: 'var(--surface-2)', color: 'var(--text)', border: '1px solid var(--border)', opacity: approve.isPending ? 0.7 : 1 }}
                  >
                    {approve.isPending
                      ? <span className="flex items-center gap-2"><Loader size={14} className="animate-spin" /> Marking reviewed...</span>
                      : <span className="flex items-center gap-2"><CheckCircle size={14} /> Mark as reviewed</span>
                    }
                  </button>
                )}

                <button
                  onClick={() => decline.mutate()}
                  disabled={decline.isPending}
                  className="flex items-center gap-2 px-5 py-2.5 rounded-lg text-sm font-medium"
                  style={{ background: 'var(--surface-2)', color: 'var(--muted)', border: '1px solid var(--border)' }}
                >
                  <XCircle size={14} />
                  Decline
                </button>
              </div>
            )}

          </div>
        )}

      </main>
    </div>
  )
}