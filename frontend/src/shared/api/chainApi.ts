import { axiosInstance } from './axiosInstance'
import { apiErrorFromAxios } from './errors'
import type { Chain } from '@entities/chain'

interface ListChainsResponse {
  chains: Chain[]
}

export const chainApi = {
  async startSearch(userId: number, wishedItemId: number): Promise<void> {
    try {
      await axiosInstance.post('/api/v1/chain', { user_id: userId, wished_item_id: wishedItemId })
    } catch (error) {
      throw apiErrorFromAxios(error)
    }
  },

  async getByWishedItemId(userId: number, wishedItemId: number): Promise<Chain[]> {
    try {
      const { data } = await axiosInstance.get<ListChainsResponse>('/api/v1/chains/by-item', {
        params: { user_id: userId, wished_item_id: wishedItemId },
      })
      return data.chains
    } catch (error) {
      throw apiErrorFromAxios(error)
    }
  },
}
