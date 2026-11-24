import { defineStore } from 'pinia'
import client from '../api/client'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('crm_token') || '',
    user: JSON.parse(localStorage.getItem('crm_user') || 'null')
  }),
  actions: {
    async login(username, password) {
      const data = await client.post('/auth/login', { username, password })
      this.token = data.token
      this.user = data.user
      localStorage.setItem('crm_token', data.token)
      localStorage.setItem('crm_user', JSON.stringify(data.user))
    },
    logout() {
      this.token = ''
      this.user = null
      localStorage.removeItem('crm_token')
      localStorage.removeItem('crm_user')
    }
  }
})
