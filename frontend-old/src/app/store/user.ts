import { defineStore } from 'pinia'

import * as authApi from '@/features/auth/api/auth'
import {
  clearToken,
  createSessionId,
  getSessionId,
  getToken,
  setSessionId,
  setSessionUser,
  setToken,
  setTokenExpiresAt
} from '@/shared/utils/auth'

export const useUserStore = defineStore('user', {
  state: () => ({
    token: getToken() as string | null,
    sessionId: getSessionId() as string | null,
    me: null as authApi.UserMe | null,
    loadingMe: false
  }),
  getters: {
    isAuthed: (s) => Boolean(s.token),
    permissions: (s) => s.me?.permissions ?? [],
    roles: (s) => s.me?.roles ?? []
  },
  actions: {
    async login(username: string, password: string) {
      const resp = await authApi.login({ username, password })
      const nextSessionId = createSessionId()
      this.token = resp.access_token
      this.sessionId = nextSessionId
      setToken(resp.access_token)
      const expiresInSec = Number(resp.expires_in ?? 0)
      setTokenExpiresAt(expiresInSec > 0 ? Date.now() + expiresInSec * 1000 : null)
      setSessionId(nextSessionId)
      setSessionUser({ id: resp.user.id, username: resp.user.username })
      this.me = resp.user
    },
    logoutLocal() {
      this.token = null
      this.sessionId = null
      this.me = null
      clearToken()
    },
    async logout() {
      try {
        await authApi.logout()
      } finally {
        this.logoutLocal()
      }
    },
    async fetchMe() {
      if (!this.token) {
        this.me = null
        return
      }
      this.loadingMe = true
      try {
        this.me = await authApi.getMe()
      } finally {
        this.loadingMe = false
      }
    },
    async changePassword(oldPassword: string, newPassword: string) {
      await authApi.changePassword(oldPassword, newPassword)
    }
  }
})
