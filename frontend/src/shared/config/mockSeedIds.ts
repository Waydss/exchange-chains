const SEED_ID_BASE = 100000

export const mockSeedIds = {
  item1: (userId: number) => userId * SEED_ID_BASE + 1,
  item2: (userId: number) => userId * SEED_ID_BASE + 2,
  wish1: (userId: number) => userId * SEED_ID_BASE + 3,
  wish2: (userId: number) => userId * SEED_ID_BASE + 4,
  chain1: (userId: number) => userId * SEED_ID_BASE + 5,
  chain2: (userId: number) => userId * SEED_ID_BASE + 6,
}
