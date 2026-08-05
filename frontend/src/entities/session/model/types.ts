export interface PublicUser {
  id: number
  login: string
  name: string
  public_rating: number
  created_at: string
}

export interface AuthResult {
  user: PublicUser
  access_token: string
  token_type: string
  expires_in: number
}

export interface RegisterInput {
  login: string
  password: string
  name: string
}

export interface LoginInput {
  login: string
  password: string
}
