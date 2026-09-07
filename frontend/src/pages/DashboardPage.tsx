import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { Shield, RefreshCw, Lock, Unlock, AlertTriangle } from 'lucide-react'
import { api } from '../api'
import type { Repository } from '../types'

// Severity score color based on risk score number
function getRiskColor(score: number): string {
  if (score >= 80) return 'var(--safe)'
  if (score >= 50) return 'var(--medium)'
  if (score >= 20) return 'var(--high)'
  return 'var(--critical)'
}

// Single repository card shown in the grid
function RepositoryCard({ repo }: { repo: Repository }) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  // Mutation to toggle monitoring on/off
  const toggleMonitor = useMutation({
    mutationFn: () =>
      repo.Monitored
        ? api.deselectRepository(repo.ID)
        : api.selectRepository(repo.ID),
    onSuccess: () => {
      // Refresh the repository list after toggling
      queryClient.invalidateQueries({ queryKey: ['repositories'] })
    },
  })

  return (
    <div
      className="p-5 rounded-lg cursor-pointer transition-all hover:border-white/20"
      style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}
      onClick={() => repo.Monitored && navigate(`/repositories/${repo.ID}`)}
    >
      {/* Repo header */}
      <div className="flex items-start justify-between mb-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            {repo.Private && (
              <Lock size={12} style={{ color: 'var(--muted)' }} />
            )}
            <span className="text-sm font-medium truncate" style={{ color: 'var(--text)' }}>
              {repo.Name}
            </span>
          </div>
          <span className="text-xs" style={{ color: 'var(--muted)' }}>
            {repo.Owner}
          </span>
        </div>

        {/* Monitor toggle button */}
        <button
          onClick={(e) => {
            e.stopPropagation() // prevent card click when clicking button
            toggleMonitor.mutate()
          }}
          className="flex items-center gap-1.5 px-2.5 py-1 rounded text-xs font-medium transition-all ml-3 shrink-0"
          style={{
            background: repo.Monitored ? 'rgba(255,255,255,0.08)' : 'var(--surface-2)',
            color: repo.Monitored ? 'var(--text)' : 'var(--muted)',
            border: `1px solid ${repo.Monitored ? 'var(--border)' : 'transparent'}`,
          }}
        >
          {repo.Monitored ? <Unlock size={11} /> : <Lock size={11} />}
          {repo.Monitored ? 'Monitoring' : 'Monitor'}
        </button>
      </div>

      {/* Only show risk info for monitored repos */}
      {repo.Monitored && (
        <div className="flex items-center gap-2 mt-3 pt-3" style={{ borderTop: '1px solid var(--border)' }}>
          <AlertTriangle size={12} style={{ color: 'var(--muted)' }} />
          <span className="text-xs" style={{ color: 'var(--muted)' }}>
            Click to view findings
          </span>
        </div>
      )}
    </div>
  )
}

export default function DashboardPage() {
  const navigate = useNavigate()

  // Fetch all repositories - this also syncs them from GitHub
  const { data: repos, isLoading, error, refetch } = useQuery({
    queryKey: ['repositories'],
    queryFn: api.getRepositories,
  })

  const monitored = repos?.filter(r => r.Monitored) ?? []
  const unmonitored = repos?.filter(r => !r.Monitored) ?? []

  return (
    <div className="min-h-screen" style={{ background: 'var(--background)' }}>

      {/* Nav */}
      <nav className="flex items-center justify-between px-8 py-4 border-b" style={{ borderColor: 'var(--border)' }}>
        <div className="flex items-center gap-2">
          <Shield size={18} style={{ color: 'var(--text)' }} />
          <span className="text-sm font-semibold" style={{ color: 'var(--text)' }}>Security Copilot</span>
        </div>
        <div className="flex items-center gap-4">
          <button
            onClick={() => navigate('/remediations')}
            className="text-xs transition-all"
            style={{ color: 'var(--muted)' }}
          >
            Remediations
          </button>
          <button
            onClick={() => refetch()}
            className="flex items-center gap-1.5 text-xs transition-all"
            style={{ color: 'var(--muted)' }}
          >
            <RefreshCw size={12} />
            Sync
          </button>
        </div>
      </nav>

      <main className="px-8 py-8 max-w-6xl mx-auto">

        {/* Header */}
        <div className="mb-8">
          <h1 className="text-2xl font-semibold mb-1" style={{ color: 'var(--text)' }}>
            Repositories
          </h1>
          <p className="text-sm" style={{ color: 'var(--muted)' }}>
            {repos?.length ?? 0} repositories · {monitored.length} monitored
          </p>
        </div>

        {/* Loading state */}
        {isLoading && (
          <div className="flex items-center justify-center py-20">
            <p className="text-sm" style={{ color: 'var(--muted)' }}>
              Syncing repositories from GitHub...
            </p>
          </div>
        )}

        {/* Error state */}
        {error && (
          <div className="p-4 rounded-lg mb-6" style={{ background: 'rgba(255,68,68,0.1)', border: '1px solid rgba(255,68,68,0.2)' }}>
            <p className="text-sm" style={{ color: 'var(--critical)' }}>
              Failed to load repositories. Make sure your session is valid.
            </p>
          </div>
        )}

        {/* Monitored repos section */}
        {monitored.length > 0 && (
          <div className="mb-8">
            <h2 className="text-xs font-medium uppercase tracking-wider mb-4" style={{ color: 'var(--muted)' }}>
              Monitored
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
              {monitored.map(repo => (
                <RepositoryCard key={repo.ID} repo={repo} />
              ))}
            </div>
          </div>
        )}

        {/* All repos section */}
        {unmonitored.length > 0 && (
          <div>
            <h2 className="text-xs font-medium uppercase tracking-wider mb-4" style={{ color: 'var(--muted)' }}>
              All repositories
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
              {unmonitored.map(repo => (
                <RepositoryCard key={repo.ID} repo={repo} />
              ))}
            </div>
          </div>
        )}
      </main>
    </div>
  )
}