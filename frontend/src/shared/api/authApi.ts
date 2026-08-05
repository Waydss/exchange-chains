import { ApiError } from './errors'
import { storage } from '@shared/lib/storage'
import type { AuthResult, LoginInput, PublicUser, RegisterInput } from '@entities/session'

// пока мок, позже интегрирую с бэком, контракт 1в1 повторяет бэк

const USERS_DB_KEY = 'mock_users_db'
const ACCESS_TOKEN_TTL_SECONDS = 86400

interface StoredUser extends PublicUser {
  password: string
}

interface TokenPayload {
  sub: number
  login: string
  iat: number
  exp: number
}

function delay(ms = 400): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function readUsers(): StoredUser[] {
  return storage.get<StoredUser[]>(USERS_DB_KEY) ?? []
}

function writeUsers(users: StoredUser[]): void {
  storage.set(USERS_DB_KEY, users)
}

function toPublicUser(user: StoredUser): PublicUser {
  return {
    id: user.id,
    login: user.login,
    name: user.name,
    public_rating: user.public_rating,
    created_at: user.created_at,
  }
}

function generateToken(user: PublicUser): string {
  const nowSeconds = Math.floor(Date.now() / 1000)
  const payload: TokenPayload = {
    sub: user.id,
    login: user.login,
    iat: nowSeconds,
    exp: nowSeconds + ACCESS_TOKEN_TTL_SECONDS,
  }

  return `${btoa(JSON.stringify(payload))}.mock-signature`
}

function parseToken(accessToken: string): TokenPayload | null {
  const [payloadPart] = accessToken.split('.')
  if (!payloadPart) return null

  try {
    return JSON.parse(atob(payloadPart)) as TokenPayload
  } catch {
    return null
  }
}

function buildAuthResult(user: PublicUser): AuthResult {
  return {
    user,
    access_token: generateToken(user),
    token_type: 'Bearer',
    expires_in: ACCESS_TOKEN_TTL_SECONDS,
  }
}

export const authApi = {
  async register(input: RegisterInput): Promise<AuthResult> {
    await delay()

    const users = readUsers()
    if (users.some((user) => user.login === input.login)) {
      throw new ApiError('login_already_exists', 'login is already taken')
    }

    const newUser: StoredUser = {
      id: users.reduce((maxId, user) => Math.max(maxId, user.id), 0) + 1,
      login: input.login,
      name: input.name,
      password: input.password,
      public_rating: 0,
      created_at: new Date().toISOString(),
    }
    writeUsers([...users, newUser])

    return buildAuthResult(toPublicUser(newUser))
  },

  async login(input: LoginInput): Promise<AuthResult> {
    await delay()

    const user = readUsers().find((candidate) => candidate.login === input.login)
    if (!user || user.password !== input.password) {
      throw new ApiError('invalid_credentials', 'invalid login or password')
    }

    return buildAuthResult(toPublicUser(user))
  },

  async me(accessToken: string): Promise<PublicUser> {
    await delay()

    const payload = parseToken(accessToken)
    if (!payload || payload.exp * 1000 < Date.now()) {
      throw new ApiError('invalid_token', 'invalid or expired access token')
    }

    const user = readUsers().find((candidate) => candidate.id === payload.sub)
    if (!user) {
      throw new ApiError('user_not_found', 'user not found')
    }

    return toPublicUser(user)
  },
}
