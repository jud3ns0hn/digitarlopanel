import { defineStore } from 'pinia'
import http, { clearToken, getToken, setToken } from '../api/client'

interface User {
  id: number
  username: string
  role: string
  two_fa_enabled?: boolean
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as User | null,
    token: getToken(),
  }),
  getters: {
    isAuthenticated: (state) => !!state.token,
    isAdmin: (state) => state.user?.role === 'admin',
    canWrite: (state) => state.user?.role === 'admin' || state.user?.role === 'operator',
  },
  actions: {
    // login returns true on success, or false when a TOTP code is still required.
    async login(username: string, password: string, code?: string): Promise<boolean> {
      const { data } = await http.post('/login', { username, password, code })
      if (data.two_factor_required) {
        return false
      }
      setToken(data.token)
      this.token = data.token
      this.user = data.user
      return true
    },
    async fetchMe() {
      const { data } = await http.get('/me')
      this.user = data
    },
    async logout() {
      try {
        await http.post('/logout')
      } catch {
        /* ignore */
      }
      clearToken()
      this.token = null
      this.user = null
    },
  },
})
