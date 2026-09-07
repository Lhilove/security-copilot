import axios from 'axios'

// All API calls go through /api which Vite proxies to localhost:8080
const client = axios.create({
  baseURL: '/api/v1',
})

// Attach JWT to every request automatically
client.interceptors.request.use((config) => {
  const token = sessionStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Redirect to home on 401
client.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      sessionStorage.removeItem('token')
      window.location.href = '/'
    }
    return Promise.reject(error)
  }
)

export default client