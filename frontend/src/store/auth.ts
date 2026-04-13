import { create } from 'zustand'
import { apiClient } from '@/api/client'
import type { User, LoginResponse } from '@/types'

interface AuthState {
  token: string | null
  user: User | null
  isLoading: boolean
  error: string | null
  login: (username: string, password: string) => Promise<void>
  logout: () => void
  hydrate: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  token: null,
  user: null,
  isLoading: false,
  error: null,

  hydrate: () => {
    const token = localStorage.getItem('openote_token')
    const userStr = localStorage.getItem('openote_user')
    if (token && userStr) {
      try {
        const user = JSON.parse(userStr) as User
        set({ token, user })
      } catch {
        localStorage.removeItem('openote_token')
        localStorage.removeItem('openote_user')
      }
    }
  },

  login: async (username: string, password: string) => {
    set({ isLoading: true, error: null })
    try {
      const { data } = await apiClient.post<LoginResponse>('/auth/login', {
        username,
        password,
      })
      localStorage.setItem('openote_token', data.accessToken)
      localStorage.setItem('openote_user', JSON.stringify(data.user))
      set({ token: data.accessToken, user: data.user, isLoading: false })
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : 'Login failed'
      set({ isLoading: false, error: message })
      throw err
    }
  },

  logout: () => {
    localStorage.removeItem('openote_token')
    localStorage.removeItem('openote_user')
    set({ token: null, user: null })
    window.location.href = '/login'
  },
}))
