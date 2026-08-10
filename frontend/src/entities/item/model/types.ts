export type Condition = 'new' | 'like_new' | 'good' | 'fair' | 'poor'

export type Status = 'searching' | 'in_chain' | 'reserved' | 'in_transit' | 'exchanged' | 'archived'

export interface CategoryResponse {
  id: number
  name: string
}

export interface PhotoResponse {
  id: number
  url: string
}

export interface Item {
  id: number
  user_id?: number
  title: string
  description: string
  condition: Condition
  category: CategoryResponse
  price_min: number
  price_max: number
  wanted_category: CategoryResponse
  photos: PhotoResponse[]
  status: Status
  created_at: string
  updated_at?: string
}

export interface CreateItemInput {
  title: string
  description: string
  condition: Condition
  category_name: string
  price_min: number
  price_max: number
  wanted_category_name: string
  wanted_title_keywords: string
  photos: File[]
}
