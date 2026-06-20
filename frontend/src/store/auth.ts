import { defineStore } from 'pinia'
import http, { clearToken, getToken, setToken } from '../api/client'

interface User {
  id: number
  username: string
  role: string
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as User | null,
    token: getToken(),
  }),
  getters: {
    isAuthenticated: (state) => !!state.token,
  },
  actions: {
    async login(username: string, password: string) {
      const { data } = await http.post('/login', { username, password })
      setToken(data.token)
      this.token = data.token
      this.user = data.user
    },
    async fetchMe() {
      const { data } = await http.get('/me')
      this.user = data
    },
    logout() {
      clearToken()
      this.token = null
      this.user = null
    },
  },
})
