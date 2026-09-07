import { useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { Shield } from 'lucide-react'

export default function CallbackPage() {
  const [searchParams] = useSearchParams()
  const { setToken } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    const token = searchParams.get('token')

    // Debug: log what we received
    console.log('Callback URL params:', window.location.search)
    console.log('Token found:', token ? 'yes' : 'no')

    if (token) {
      setToken(token)
      navigate('/dashboard', { replace: true })
    } else {
      // Try reading directly from URL as fallback
      const urlParams = new URLSearchParams(window.location.search)
      const urlToken = urlParams.get('token')
      console.log('Fallback token:', urlToken ? 'yes' : 'no')

      if (urlToken) {
        setToken(urlToken)
        navigate('/dashboard', { replace: true })
      } else {
        navigate('/', { replace: true })
      }
    }
  }, [])

  return (
    <div
      className="min-h-screen flex flex-col items-center justify-center"
      style={{ background: 'var(--background)' }}
    >
      <Shield size={32} style={{ color: 'var(--text)' }} className="mb-4" />
      <p className="text-sm" style={{ color: 'var(--muted)' }}>
        Connecting your GitHub account...
      </p>
    </div>
  )
}