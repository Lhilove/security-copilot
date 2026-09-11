import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, Shield, CheckCircle, XCircle, GitPullRequest, Loader } from 'lucide-react'
import { api } from '../api'
import type { AnalysisResult } from '../types'
import NotificationBell from '../components/NotificationBell'

export default function FindingPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [analysis, setAnalysis] = useState<AnalysisResult | null>(null)
  const [prUrl, setPrUrl] = useState<string | null>(null)

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
    onSuccess: () => setTimeout(() => createPR.mutate(), 500),
  })

  const decline = useMutation({
    mutationFn: () => api.declineFinding(id!),
    onSuccess: () => navigate(-1),
  })

  const isApproving = approve.isPending || createPR.isPending

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

        {!analysis && !analyze.isPending && (
          <div className="p-8 rounded-lg text-center" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
            <Shield size={32} className="mx-auto mb-4" style={{ color: 'var(--muted)' }} />
            <p className="text-sm mb-2" style={{ color: 'var(--text)' }}>Analyze this finding with AI</p>
            <p className="text-xs mb-6" style={{ color: 'var(--muted)' }}>DeepSeek will explain the business impact and propose a fix</p>
            {analyze.isError && (
              <p className="text-xs mb-4" style={{ color: 'var(--critical)' }}>Analysis failed. Try again.</p>
            )}
            <button onClick={() => analyze.mutate()} className="px-5 py-2.5 rounded-lg text-sm font-medium" style={{ background: 'var(--accent)', color: '#000' }}>
              Analyze finding
            </button>
          </div>
        )}

        {analyze.isPending && (
          <div className="p-8 rounded-lg text-center" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
            <Loader size={24} className="mx-auto mb-4 animate-spin" style={{ color: 'var(--muted)' }} />
            <p className="text-sm" style={{ color: 'var(--muted)' }}>Analyzing with DeepSeek AI...</p>
            <p className="text-xs mt-2" style={{ color: 'var(--muted)' }}>This may take a minute</p>
          </div>
        )}

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

            {analysis.proposed_code && (
              <div className="p-5 rounded-lg" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
                <p className="text-xs font-medium uppercase tracking-wider mb-3" style={{ color: 'var(--muted)' }}>Proposed fix</p>
                <pre className="text-xs leading-relaxed overflow-x-auto p-4 rounded" style={{ background: 'var(--surface-2)', color: 'var(--text)', fontFamily: 'monospace' }}>
                  {analysis.proposed_code}
                </pre>
              </div>
            )}

            {createPR.isError && (
              <div className="p-4 rounded-lg" style={{ background: 'rgba(255,68,68,0.1)', border: '1px solid rgba(255,68,68,0.2)' }}>
                <p className="text-xs" style={{ color: 'var(--critical)' }}>PR creation failed. You can apply the fix manually.</p>
              </div>
            )}

            {!prUrl && (
              <div className="flex gap-3 pt-2">
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