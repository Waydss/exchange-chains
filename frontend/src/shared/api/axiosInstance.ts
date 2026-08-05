import axios from 'axios'
import { env } from '@shared/config/env'

export const axiosInstance = axios.create({
  baseURL: env.apiUrl,
})
