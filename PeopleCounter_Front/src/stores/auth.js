import { defineStore } from 'pinia'
import api from '../api'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null
  }),

  getters: {
    isAdmin: (state) => state.user?.role === 'admin',
    isAuthenticated: (state) => state.user !== null
  },

  actions: {
    async fetchUser() {
      try {
        const userData = await api.getCurrentUser()
        this.user = userData
        return userData
      } catch (error) {
        this.user = null
        throw error
      }
    },

    async login(loginStr, passwordStr) {
      await api.login(loginStr, passwordStr)
      await this.fetchUser()
    },

    logout() {
      this.user = null
      localStorage.removeItem('accessToken')
      localStorage.removeItem('refreshToken')
    }
  }
})