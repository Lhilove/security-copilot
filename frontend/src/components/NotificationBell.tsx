import { useState, useRef, useEffect } from 'react'
import { Bell } from 'lucide-react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'
import type { Notification } from '../types'

export default function NotificationBell() {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const { data } = useQuery({
    queryKey: ['notifications'],
    queryFn: api.getNotifications,
    refetchInterval: 30000, // poll every 30 seconds
  })

  const markRead = useMutation({
    mutationFn: api.markNotificationRead,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['notifications'] }),
  })

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  const notifications = data?.notifications ?? []
  const unreadCount = data?.unread_count ?? 0

  function handleNotificationClick(n: Notification) {
    if (!n.Read) markRead.mutate(n.ID)
    setOpen(false)
    if (n.FindingID) navigate(`/findings/${n.FindingID}`)
  }

  return (
    <div ref={ref} className="relative">
      {/* Bell button */}
      <button
        onClick={() => setOpen(prev => !prev)}
        className="relative flex items-center justify-center w-8 h-8 rounded-lg transition-all"
        style={{ color: 'var(--muted)' }}
      >
        <Bell size={16} />
        {unreadCount > 0 && (
          <span
            className="absolute -top-1 -right-1 flex items-center justify-center w-4 h-4 rounded-full text-white"
            style={{ background: 'var(--critical)', fontSize: '10px', fontWeight: 600 }}
          >
            {unreadCount > 9 ? '9+' : unreadCount}
          </span>
        )}
      </button>

      {/* Dropdown */}
      {open && (
        <div
          className="absolute right-0 mt-2 w-80 rounded-lg shadow-xl z-50 overflow-hidden"
          style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}
        >
          {/* Header */}
          <div
            className="flex items-center justify-between px-4 py-3 border-b"
            style={{ borderColor: 'var(--border)' }}
          >
            <span className="text-xs font-semibold" style={{ color: 'var(--text)' }}>
              Notifications
            </span>
            {unreadCount > 0 && (
              <span className="text-xs" style={{ color: 'var(--muted)' }}>
                {unreadCount} unread
              </span>
            )}
          </div>

          {/* List */}
          <div className="max-h-80 overflow-y-auto">
            {notifications.length === 0 ? (
              <div className="flex items-center justify-center py-8">
                <p className="text-xs" style={{ color: 'var(--muted)' }}>
                  No notifications yet
                </p>
              </div>
            ) : (
              notifications.map(n => (
                <button
                  key={n.ID}
                  onClick={() => handleNotificationClick(n)}
                  className="w-full text-left px-4 py-3 border-b transition-all hover:opacity-80"
                  style={{
                    borderColor: 'var(--border)',
                    background: n.Read ? 'transparent' : 'rgba(255,255,255,0.03)',
                  }}
                >
                  <div className="flex items-start gap-2">
                    {!n.Read && (
                      <span
                        className="mt-1.5 w-1.5 h-1.5 rounded-full shrink-0"
                        style={{ background: 'var(--critical)' }}
                      />
                    )}
                    <div className={!n.Read ? '' : 'ml-3.5'}>
                      <p className="text-xs font-medium mb-0.5" style={{ color: 'var(--text)' }}>
                        {n.Title}
                      </p>
                      <p className="text-xs line-clamp-2" style={{ color: 'var(--muted)' }}>
                        {n.Body}
                      </p>
                      <p className="text-xs mt-1" style={{ color: 'var(--muted)', opacity: 0.6 }}>
                        {new Date(n.CreatedAt).toLocaleString()}
                      </p>
                    </div>
                  </div>
                </button>
              ))
            )}
          </div>

          {/* Footer */}
          <div
            className="px-4 py-2 border-t"
            style={{ borderColor: 'var(--border)' }}
          >
            <button
              onClick={() => { setOpen(false); navigate('/settings') }}
              className="text-xs transition-all"
              style={{ color: 'var(--muted)' }}
            >
              Notification settings
            </button>
          </div>
        </div>
      )}
    </div>
  )
}