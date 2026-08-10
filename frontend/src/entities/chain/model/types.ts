export type ChainStatus =
  'matched' | 'pending' | 'locked' | 'in_delivery' | 'completed' | 'cancelled' | 'disputed'

export type ParticipantStatus = 'pending' | 'accepted' | 'declined' | 'expired'

export type BarterStatus =
  'pending' | 'method_selected' | 'given' | 'received' | 'rejected' | 'returned'

export interface GiveItem {
  id: number
  user_id: number
  title: string
  category_id: number
  price_min: number
  price_max: number
  status: string
}

export interface WishRef {
  id: number
  user_id: number
  category_id: number
  price_min: number
  price_max: number
  title_keywords: string
}

export interface Barter {
  id: number
  position: number
  from_user_id: number
  to_user_id: number
  give: GiveItem
  wish: WishRef
  status: BarterStatus
}

export interface ChainParticipant {
  user_id: number
  status: ParticipantStatus
  accepted_at?: string
  declined_at?: string
  updated_at: string
}

export interface Chain {
  id: number
  root_id: number
  root_wish: WishRef
  signature: string
  status: ChainStatus
  score: number
  expires_at?: string
  locked_at?: string
  completed_at?: string
  cancelled_at?: string
  cancelled_by_user_id?: number
  cancel_reason?: string
  created_at: string
  participants: ChainParticipant[]
  barters: Barter[]
}
