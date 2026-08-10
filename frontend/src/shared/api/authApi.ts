import { axiosInstance } from './axiosInstance'
import { apiErrorFromAxios } from './errors'
import type { AuthResult, LoginInput, PublicUser, RegisterInput } from '@entities/session'

export const authApi = {
  async register(input: RegisterInput): Promise<AuthResult> {
    try {
      const { data } = await axiosInstance.post<AuthResult>('/api/v1/auth/register', input)
      return data
    } catch (error) {
      throw apiErrorFromAxios(error)
    }
  },

  async login(input: LoginInput): Promise<AuthResult> {
    try {
      const { data } = await axiosInstance.post<AuthResult>('/api/v1/auth/login', input)
      return data
    } catch (error) {
      throw apiErrorFromAxios(error)
    }
  },

  async me(): Promise<PublicUser> {
    try {
      const { data } = await axiosInstance.get<PublicUser>('/api/v1/me')
      return data
    } catch (error) {
      throw apiErrorFromAxios(error)
    }
  },
}
