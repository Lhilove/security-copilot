import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowLeft, Shield, Mail, MessageSquare, Send, Hash } from 'lucide-react'
import { useQuery, useMutation } from '@tanstack/react-query'
import { api } from '../api'
import type { NotificationSettings } from '../types'
import NotificationBell from '../components/NotificationBell'

const defaultSettings: NotificationSettings = {
  email: '',
  slack_webhook_url: '',
  discord_webhook_url: '',
  telegram_bot_token: '',
  telegram_chat_id: '',
  email_enabled: false,
  slack_enabled: false,
  discord_enabled: false,
  telegram_enabled: false,
}

function ChannelSection({
  title,
  icon,
  enabled,
  onToggle,
  children,
}: {
  title: string
  icon: React.ReactNode
  enabled: boolean
  onToggle: () => void
  children: React.ReactNode
}) {
  return (
    <div
      className="p-5 rounded-lg"
      style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}
    >
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          {icon}
          <span className="text-sm font-medium" style={{ color: 'var(--text)' }}>
            {title}
          </span>
        </div>
        {/* Toggle */}
        <button
          onClick={onToggle}
          className="relative w-10 h-5 rounded-full transition-all"
          style={{ background: enabled ? 'var(--critical)' : 'var(--surface-2)' }}
        >
          <span
            className="absolute top-0.5 w-4 h-4 rounded-full bg-white transition-all"
            style={{ left: enabled ? '22px' : '2px' }}
          />
        </button>
      </div>
      {enabled && (
        <div className="space-y-3">
          {children}
        </div>
      )}
    </div>
  )
}

function Input({
  label,
  value,
  onChange,
  placeholder,
  type = 'text',
}: {
  label: string
  value: string
  onChange: (v: string) => void
  placeholder?: string
  type?: string
}) {
  return (
    <div>
      <label className="block text-xs mb-1.5" style={{ color: 'var(--muted)' }}>
        {label}
      </label>
      <input
        type={type}
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        className="w-full px-3 py-2 rounded-lg text-sm outline-none transition-all"
        style={{
          background: 'var(--surface-2)',
          border: '1px solid var(--border)',
          color: 'var(--text)',
        }}
      />
    </div>
  )
}

export default function SettingsPage() {
  const navigate = useNavigate()
  const [settings, setSettings] = useState<NotificationSettings>(defaultSettings)
  const [saved, setSaved] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['notification-settings'],
    queryFn: api.getNotificationSettings,
  })

  useEffect(() => {
    if (data) setSettings(data)
  }, [data])

  const save = useMutation({
    mutationFn: api.saveNotificationSettings,
    onSuccess: () => {
      setSaved(true)
      setTimeout(() => setSaved(false), 3000)
    },
  })

  function update(patch: Partial<NotificationSettings>) {
    setSettings(prev => ({ ...prev, ...patch }))
  }

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
            Settings
          </span>
        </div>
        <NotificationBell />
      </nav>

      <main className="px-8 py-8 max-w-2xl mx-auto">

        <div className="mb-8">
          <h1 className="text-2xl font-semibold mb-1" style={{ color: 'var(--text)' }}>
            Notification Settings
          </h1>
          <p className="text-sm" style={{ color: 'var(--muted)' }}>
            Get alerted when new findings arrive. Approve or decline fixes directly from any channel.
          </p>
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <p className="text-sm" style={{ color: 'var(--muted)' }}>Loading settings...</p>
          </div>
        ) : (
          <div className="space-y-4">

            {/* Email */}
            <ChannelSection
              title="Email"
              icon={<Mail size={15} style={{ color: 'var(--muted)' }} />}
              enabled={settings.email_enabled}
              onToggle={() => update({ email_enabled: !settings.email_enabled })}
            >
              <Input
                label="Email address"
                value={settings.email}
                onChange={v => update({ email: v })}
                placeholder="you@example.com"
                type="email"
              />
            </ChannelSection>

            {/* Slack */}
            <ChannelSection
              title="Slack"
              icon={<Hash size={15} style={{ color: 'var(--muted)' }} />}
              enabled={settings.slack_enabled}
              onToggle={() => update({ slack_enabled: !settings.slack_enabled })}
            >
              <Input
                label="Webhook URL"
                value={settings.slack_webhook_url}
                onChange={v => update({ slack_webhook_url: v })}
                placeholder="https://hooks.slack.com/services/..."
              />
            </ChannelSection>

            {/* Discord */}
            <ChannelSection
              title="Discord"
              icon={<MessageSquare size={15} style={{ color: 'var(--muted)' }} />}
              enabled={settings.discord_enabled}
              onToggle={() => update({ discord_enabled: !settings.discord_enabled })}
            >
              <Input
                label="Webhook URL"
                value={settings.discord_webhook_url}
                onChange={v => update({ discord_webhook_url: v })}
                placeholder="https://discord.com/api/webhooks/..."
              />
            </ChannelSection>

            {/* Telegram */}
            <ChannelSection
              title="Telegram"
              icon={<Send size={15} style={{ color: 'var(--muted)' }} />}
              enabled={settings.telegram_enabled}
              onToggle={() => update({ telegram_enabled: !settings.telegram_enabled })}
            >
              <Input
                label="Bot Token"
                value={settings.telegram_bot_token}
                onChange={v => update({ telegram_bot_token: v })}
                placeholder="123456:ABC-DEF..."
              />
              <Input
                label="Chat ID"
                value={settings.telegram_chat_id}
                onChange={v => update({ telegram_chat_id: v })}
                placeholder="-1001234567890"
              />
            </ChannelSection>

            {/* Save button */}
            <div className="flex items-center justify-between pt-2">
              {saved && (
                <p className="text-xs" style={{ color: 'var(--low)' }}>
                  Settings saved.
                </p>
              )}
              {!saved && <div />}
              <button
                onClick={() => save.mutate(settings)}
                disabled={save.isPending}
                className="px-5 py-2 rounded-lg text-sm font-medium transition-all"
                style={{
                  background: save.isPending ? 'var(--surface-2)' : 'var(--text)',
                  color: save.isPending ? 'var(--muted)' : 'var(--background)',
                }}
              >
                {save.isPending ? 'Saving...' : 'Save settings'}
              </button>
            </div>

          </div>
        )}
      </main>
    </div>
  )
}