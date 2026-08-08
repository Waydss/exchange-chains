import type { Status } from './types'

interface StatusMeta {
  label: string
  background: string
  color: string
}

export const STATUS_META: Record<Status, StatusMeta> = {
  searching: { label: 'В поиске', background: '#f2f1ef', color: '#5c5c5c' },
  in_chain: { label: 'В цепочке', background: '#e5f6ff', color: '#0081f6' },
  reserved: { label: 'В процессе', background: '#fff4e0', color: '#b5760a' },
  in_transit: { label: 'В доставке', background: '#e5f6ff', color: '#0081f6' },
  exchanged: { label: 'Обменяно', background: '#e5f7ea', color: '#018a3d' },
  archived: { label: 'Архив', background: '#f2f1ef', color: '#8c8c8c' },
}
