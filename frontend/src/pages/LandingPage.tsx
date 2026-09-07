import { Shield, GitBranch, Zap, Lock, GitPullRequest } from 'lucide-react'

export default function LandingPage() {

  const handleConnectGitHub = () => {
    // Redirect to backend GitHub OAuth endpoint
    window.location.href = 'http://localhost:8080/api/v1/auth/github'
  }

  return (
    <div className="min-h-screen flex flex-col" style={{ background: 'var(--background)' }}>

      {/* Navigation */}
      <nav className="flex items-center justify-between px-8 py-5 border-b" style={{ borderColor: 'var(--border)' }}>
        <div className="flex items-center gap-2">
          <Shield size={20} style={{ color: 'var(--accent)' }} />
          <span className="font-semibold text-sm tracking-tight" style={{ color: 'var(--text)' }}>
            Security Copilot
          </span>
        </div>
        <button
          onClick={handleConnectGitHub}
          className="flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-all"
          style={{ background: 'var(--surface)', color: 'var(--text)', border: '1px solid var(--border)' }}
        >
          <GitBranch size={16} />
          Connect GitHub
        </button>
      </nav>

      {/* Hero */}
      <main className="flex-1 flex flex-col items-center justify-center px-8 text-center">

        {/* Badge */}
        <div
          className="flex items-center gap-2 px-3 py-1 rounded-full text-xs font-medium mb-8"
          style={{ background: 'var(--surface-2)', color: 'var(--muted)', border: '1px solid var(--border)' }}
        >
          <div className="w-1.5 h-1.5 rounded-full" style={{ background: 'var(--text)' }} />
          AI-powered security remediation
        </div>

        {/* Headline */}
        <h1 className="text-5xl font-semibold tracking-tight mb-6 max-w-2xl leading-tight" style={{ color: 'var(--text)' }}>
          Fix security issues
          <br />
          <span style={{ color: 'var(--accent)' }}>before they ship</span>
        </h1>

        {/* Subheadline */}
        <p className="text-lg mb-10 max-w-xl leading-relaxed" style={{ color: 'var(--muted)' }}>
          Connect your GitHub repositories. Security Copilot aggregates CodeQL,
          Dependabot, and Secret Scanning findings, explains the business impact,
          and opens a pull request with the fix. You just approve.
        </p>

        {/* CTA */}
        <button
          onClick={handleConnectGitHub}
          className="flex items-center gap-3 px-6 py-3 rounded-lg text-sm font-semibold transition-all hover:opacity-90"
          style={{ background: 'var(--accent)', color: '#000000' }}
        >
          <GitBranch size={18} />
          Connect GitHub, it's free
        </button>

        <p className="mt-4 text-xs" style={{ color: 'var(--muted)' }}>
          No credit card. No setup. Connect and scan.
        </p>

        {/* Feature grid */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-20 max-w-3xl w-full">
          {[
            {
              icon: <Zap size={18} style={{ color: 'var(--accent)' }} />,
              title: 'Instant aggregation',
              description: 'CodeQL, Dependabot, and Secret Scanning findings in one place, ordered by severity.',
            },
            {
              icon: <Lock size={18} style={{ color: 'var(--accent)' }} />,
              title: 'AI explains the risk',
              description: 'Not just "SQL injection found." Business impact, exploit scenario, and fix in plain English.',
            },
            {
              icon: <GitPullRequest size={18} style={{ color: 'var(--accent)' }} />,
              title: 'One-click PR',
              description: 'Approve the fix. Security Copilot opens the pull request. You review and merge.',
            },
          ].map((feature) => (
            <div
              key={feature.title}
              className="p-5 rounded-lg text-left"
              style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}
            >
              <div className="mb-3">{feature.icon}</div>
              <h3 className="text-sm font-semibold mb-1" style={{ color: 'var(--text)' }}>
                {feature.title}
              </h3>
              <p className="text-xs leading-relaxed" style={{ color: 'var(--muted)' }}>
                {feature.description}
              </p>
            </div>
          ))}
        </div>
      </main>

      {/* Footer */}
      <footer className="px-8 py-5 border-t text-center" style={{ borderColor: 'var(--border)' }}>
        <p className="text-xs" style={{ color: 'var(--muted)' }}>
          {'Built by '}
          <a href="https://linkedin.com/in/adewole-oluwapelumi" target="_blank" rel="noopener noreferrer" style={{ color: 'var(--accent)' }}>
            Adewole Oluwapelumi
          </a>
          {' · MIT License'}
        </p>
      </footer>

    </div>
  )
}