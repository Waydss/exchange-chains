import { axiosInstance } from './axiosInstance'
import { apiErrorFromAxios } from './errors'
import type { CreateItemInput, Item } from '@entities/item'

interface ListItemsResponse {
  items: Item[]
}

function buildFormData(input: CreateItemInput): FormData {
  const formData = new FormData()
  formData.append('title', input.title)
  formData.append('description', input.description)
  formData.append('condition', input.condition)
  formData.append('category_name', input.category_name)
  formData.append('price_min', String(input.price_min))
  formData.append('price_max', String(input.price_max))
  formData.append('wanted_category_name', input.wanted_category_name)
  formData.append('wanted_title_keywords', input.wanted_title_keywords)
  input.photos.forEach((photo) => formData.append('photos', photo))
  return formData
}

export const itemApi = {
  async listMine(): Promise<Item[]> {
    try {
      const { data } = await axiosInstance.get<ListItemsResponse>('/api/v1/items')
      return data.items
    } catch (error) {
      throw apiErrorFromAxios(error)
    }
  },

  async getById(itemId: number): Promise<Item> {
    try {
      const { data } = await axiosInstance.get<Item>(`/api/v1/items/${itemId}`)
      return data
    } catch (error) {
      throw apiErrorFromAxios(error)
    }
  },

  async create(input: CreateItemInput): Promise<Item> {
    try {
      const { data } = await axiosInstance.post<Item>('/api/v1/items', buildFormData(input), {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      return data
    } catch (error) {
      throw apiErrorFromAxios(error)
    }
  },
}
