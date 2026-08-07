import { makeAutoObservable } from 'mobx'
import { storage } from '@shared/lib/storage'
import type { AuthResult, PublicUser } from './types'

const SESSION_STORAGE_KEY = 'session'

interface PersistedSession {
  user: PublicUser
  accessToken: string
}

class SessionStore {
  user: PublicUser | null = null
  accessToken: string | null = null

  constructor() {
    makeAutoObservable(this)
    this.hydrate()
  }

  get isAuthenticated(): boolean {
    return this.user !== null && this.accessToken !== null
  }

  setSession(result: AuthResult): void {
    this.user = result.user
    this.accessToken = result.access_token
    storage.set<PersistedSession>(SESSION_STORAGE_KEY, {
      user: result.user,
      accessToken: result.access_token,
    })
  }

  clear(): void {
    this.user = null
    this.accessToken = null
    storage.remove(SESSION_STORAGE_KEY)
  }

  private hydrate(): void {
    const persisted = storage.get<PersistedSession>(SESSION_STORAGE_KEY)
    if (!persisted) return

    this.user = persisted.user
    this.accessToken = persisted.accessToken
  }
}

export const sessionStore = new SessionStore()
