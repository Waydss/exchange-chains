import { axiosInstance } from './axiosInstance'
import { apiErrorFromAxios } from './errors'
import type { CategoryResponse } from '@entities/item'

export interface Wish {
  id: number
  user_id: number
  category: CategoryResponse
  price_min: number
  price_max: number
  title_keywords: string
  created_at: string
  updated_at: string
}

interface ListWishesResponse {
  wishes: Wish[]
}

export const wishApi = {
  async listMine(): Promise<Wish[]> {
    try {
      const { data } = await axiosInstance.get<ListWishesResponse>('/api/v1/wishes')
      return data.wishes
    } catch (error) {
      throw apiErrorFromAxios(error)
    }
  },
}
