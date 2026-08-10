import type { Barter, Chain } from './types'

export function getReceivedBarter(chain: Chain, userId: number): Barter | undefined {
  return chain.barters.find((barter) => barter.to_user_id === userId)
}
