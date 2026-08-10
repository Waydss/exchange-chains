import axios from 'axios'
import { env } from '@shared/config/env'
import { sessionStore } from '@entities/session'

export const axiosInstance = axios.create({
  baseURL: env.apiUrl,
})

axiosInstance.interceptors.request.use((config) => {
  const token = sessionStore.accessToken
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

axiosInstance.interceptors.response.use(
  (response) => response,
  (error) => {
    if (axios.isAxiosError(error) && error.response?.status === 401) {
      sessionStore.clear()
    }
    return Promise.reject(error)
  },
)
