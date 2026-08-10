import { storage } from '@shared/lib/storage'
import { mockCategoryStore } from './mockCategoryStore'
import { mockSeedIds } from '@shared/config/mockSeedIds'
import type { Chain } from '@entities/chain'

// Мок будущего GET /api/v1/chains/by-item?user_id&wished_item_id
// (internal/chain_builder/handler.go GetChainsByItem), контракт повторяет ChainReadDTO.

const CHAINS_DB_KEY = 'mock_chains_db'

function delay(ms = 400): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function readChains(): Chain[] {
  return storage.get<Chain[]>(CHAINS_DB_KEY) ?? []
}

function writeChains(chains: Chain[]): void {
  storage.set(CHAINS_DB_KEY, chains)
}

// Досеивает 2 найденные цепочки для "wish1" демо-объявления (см. itemApi.ensureSeeded):
// цепочку из 3 (барабанная установка) и из 2 (клавишные Casio). Для второго демо-объявления
// (wish2) цепочек нарочно нет — оно должно показывать состояние "ищем".
function ensureSeeded(userId: number): void {
  const chains = readChains()
  if (chains.some((chain) => chain.id === mockSeedIds.chain1(userId))) return

  const drumsKeysCategory = mockCategoryStore.getOrCreate('Ударные и клавишные')
  const audioCategory = mockCategoryStore.getOrCreate('Аудиотехника')
  const now = new Date().toISOString()
  const wishedItemId = mockSeedIds.wish1(userId)

  const rootWish = {
    id: wishedItemId,
    user_id: userId,
    category_id: drumsKeysCategory.id,
    price_min: 38000,
    price_max: 55000,
    title_keywords: '',
  }

  // Условные "чужие" участники цепочек — их профили не нужны для этой задачи,
  // страница деталей цепочки (со списком участников) будет следующей таской.
  const otherUserA = -1
  const otherUserB = -2
  const otherUserC = -3

  const chainOfThree: Chain = {
    id: mockSeedIds.chain1(userId),
    root_id: userId,
    root_wish: rootWish,
    signature: `mock-chain-${mockSeedIds.chain1(userId)}`,
    status: 'matched',
    score: 0.82,
    created_at: now,
    participants: [
      { user_id: userId, status: 'pending', updated_at: now },
      { user_id: otherUserA, status: 'pending', updated_at: now },
      { user_id: otherUserB, status: 'pending', updated_at: now },
    ],
    barters: [
      {
        id: 1,
        position: 1,
        from_user_id: userId,
        to_user_id: otherUserA,
        give: {
          id: wishedItemId - 2,
          user_id: userId,
          title: 'Синтезатор Yamaha PSR',
          category_id: mockCategoryStore.getOrCreate('Музыкальные инструменты').id,
          price_min: 38000,
          price_max: 55000,
          status: 'searching',
        },
        wish: { ...rootWish, id: wishedItemId + 1000, user_id: otherUserA },
        status: 'pending',
      },
      {
        id: 2,
        position: 2,
        from_user_id: otherUserA,
        to_user_id: otherUserB,
        give: {
          id: wishedItemId + 1001,
          user_id: otherUserA,
          title: 'Микшер Behringer',
          category_id: audioCategory.id,
          price_min: 28000,
          price_max: 32000,
          status: 'searching',
        },
        wish: { ...rootWish, id: wishedItemId + 1002, user_id: otherUserB },
        status: 'pending',
      },
      {
        id: 3,
        position: 3,
        from_user_id: otherUserB,
        to_user_id: userId,
        give: {
          id: wishedItemId + 1003,
          user_id: otherUserB,
          title: 'Барабанная установка',
          category_id: drumsKeysCategory.id,
          price_min: 48000,
          price_max: 50000,
          status: 'searching',
        },
        wish: rootWish,
        status: 'pending',
      },
    ],
  }

  const chainOfTwo: Chain = {
    id: mockSeedIds.chain2(userId),
    root_id: userId,
    root_wish: rootWish,
    signature: `mock-chain-${mockSeedIds.chain2(userId)}`,
    status: 'matched',
    score: 0.74,
    created_at: now,
    participants: [
      { user_id: userId, status: 'pending', updated_at: now },
      { user_id: otherUserC, status: 'pending', updated_at: now },
    ],
    barters: [
      {
        id: 4,
        position: 1,
        from_user_id: userId,
        to_user_id: otherUserC,
        give: {
          id: wishedItemId - 2,
          user_id: userId,
          title: 'Синтезатор Yamaha PSR',
          category_id: mockCategoryStore.getOrCreate('Музыкальные инструменты').id,
          price_min: 38000,
          price_max: 55000,
          status: 'searching',
        },
        wish: { ...rootWish, id: wishedItemId + 2000, user_id: otherUserC },
        status: 'pending',
      },
      {
        id: 5,
        position: 2,
        from_user_id: otherUserC,
        to_user_id: userId,
        give: {
          id: wishedItemId + 2001,
          user_id: otherUserC,
          title: 'Клавишные Casio',
          category_id: drumsKeysCategory.id,
          price_min: 42000,
          price_max: 44000,
          status: 'searching',
        },
        wish: rootWish,
        status: 'pending',
      },
    ],
  }

  writeChains([...chains, chainOfThree, chainOfTwo])
}

export const chainApi = {
  async getByWishedItemId(userId: number, wishedItemId: number): Promise<Chain[]> {
    await delay()
    ensureSeeded(userId)

    return readChains().filter((chain) => chain.root_id === userId && chain.root_wish.id === wishedItemId)
  },
}
