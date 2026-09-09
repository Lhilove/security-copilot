import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { Shield, RefreshCw, Lock, Unlock, AlertTriangle, ShieldCheck } from 'lucide-react'
import { api } from '../api'
import type { Repository } from '../types'
import NotificationBell from '../components/NotificationBell'

function RepositoryCard({ repo }: { repo: Repository }) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [scanStarted, setScanStarted] = useState(false)

  const toggleMonitor = useMutation({
    mutationFn: () =>
      repo.Monitored
        ? api.deselectRepository(repo.ID)
        : api.selectRepository(repo.ID),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['repositories'] })
    },
  })

  const setupSecurity = useMutation({
    mutationFn: () => api.setupSecurity(repo.ID),
    onSuccess: () => {
      setScanStarted(true)
    },
  })

  return (
    <div
      className="p-5 rounded-lg transition-all hover:border-white/20"
      style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}
    >
      {/* Repo header */}
      <div
        className="flex items-start justify-between mb-3 cursor-pointer"
        onClick={() => repo.Monitored && navigate(`/repositories/${repo.ID}`)}
      >
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

        {/* Monitor toggle */}
        <button
          onClick={(e) => {
            e.stopPropagation()
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

      {/* Actions for monitored repos */}
      {repo.Monitored && (
        <div
          className="mt-3 pt-3 space-y-2"
          style={{ borderTop: '1px solid var(--border)' }}
        >
          {/* Enable Security button */}
          {!scanStarted ? (
            <button
              onClick={() => setupSecurity.mutate()}
              disabled={setupSecurity.isPending}
              className="w-full flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-xs font-medium transition-all"
              style={{
                background: setupSecurity.isPending ? 'var(--surface-2)' : 'rgba(22,163,74,0.1)',
                color: setupSecurity.isPending ? 'var(--muted)' : '#16a34a',
                border: '1px solid rgba(22,163,74,0.2)',
              }}
            >
              <ShieldCheck size={12} />
              {setupSecurity.isPending ? 'Enabling security...' : 'Enable Security & Scan'}
            </button>
          ) : (
            <div
              className="w-full flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-xs"
              style={{
                background: 'rgba(22,163,74,0.05)',
                color: '#16a34a',
                border: '1px solid rgba(22,163,74,0.15)',
              }}
            >
              <ShieldCheck size={12} />
              AI scan running — check findings in a minute
            </div>
          )}

          {/* View findings link */}
          <button
            onClick={() => navigate(`/repositories/${repo.ID}`)}
            className="w-full flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-xs transition-all"
            style={{ color: 'var(--muted)' }}
          >
            <AlertTriangle size={11} />
            View findings
          </button>
        </div>
      )}
    </div>
  )
}

export default function DashboardPage() {
  const navigate = useNavigate()

  const { data: repos, isLoading, error, refetch } = useQuery({
    queryKey: ['repositories'],
    queryFn: api.getRepositories,
  })

  const monitored = repos?.filter(r => r.Monitored) ?? []
  const unmonitored = repos?.filter(r => !r.Monitored) ?? []

  return (
    <div className="min-h-screen" style={{ background: 'var(--background)' }}>

      {/* Nav */}
      <nav
        className="flex items-center justify-between px-8 py-4 border-b"
        style={{ borderColor: 'var(--border)' }}
      >
        <div className="flex items-center gap-2">
          <Shield size={18} style={{ color: 'var(--text)' }} />
          <span className="text-sm font-semibold" style={{ color: 'var(--text)' }}>
            Security Copilot
          </span>
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
            onClick={() => navigate('/settings')}
            className="text-xs transition-all"
            style={{ color: 'var(--muted)' }}
          >
            Settings
          </button>
          <button
            onClick={() => refetch()}
            className="flex items-center gap-1.5 text-xs transition-all"
            style={{ color: 'var(--muted)' }}
          >
            <RefreshCw size={12} />
            Sync
          </button>
          <NotificationBell />
        </div>
      </nav>

      <main className="px-8 py-8 max-w-6xl mx-auto">

        <div className="mb-8">
          <h1 className="text-2xl font-semibold mb-1" style={{ color: 'var(--text)' }}>
            Repositories
          </h1>
          <p className="text-sm" style={{ color: 'var(--muted)' }}>
            {repos?.length ?? 0} repositories · {monitored.length} monitored
          </p>
        </div>

        {isLoading && (
          <div className="flex items-center justify-center py-20">
            <p className="text-sm" style={{ color: 'var(--muted)' }}>
              Syncing repositories from GitHub...
            </p>
          </div>
        )}

        {error && (
          <div
            className="p-4 rounded-lg mb-6"
            style={{ background: 'rgba(255,68,68,0.1)', border: '1px solid rgba(255,68,68,0.2)' }}
          >
            <p className="text-sm" style={{ color: 'var(--critical)' }}>
              Failed to load repositories. Make sure your session is valid.
            </p>
          </div>
        )}

        {monitored.length > 0 && (
          <div className="mb-8">
            <h2
              className="text-xs font-medium uppercase tracking-wider mb-4"
              style={{ color: 'var(--muted)' }}
            >
              Monitored
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
              {monitored.map(repo => (
                <RepositoryCard key={repo.ID} repo={repo} />
              ))}
            </div>
          </div>
        )}

        {unmonitored.length > 0 && (
          <div>
            <h2
              className="text-xs font-medium uppercase tracking-wider mb-4"
              style={{ color: 'var(--muted)' }}
            >
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