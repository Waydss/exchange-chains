import { ApiError } from './errors'
import { storage } from '@shared/lib/storage'
import { mockCategoryStore } from './mockCategoryStore'
import { mockSeedIds } from '@shared/config/mockSeedIds'
import type { CreateItemInput, Item, PhotoResponse } from '@entities/item'

// пока мок, позже интегрирую с бэком (POST /api/v1/items), загрузка фото мок
// листинга вещей (GET /items) на бэке пока нет вообще, сделал моковым.

const ITEMS_DB_KEY = 'mock_items_db'
const MAX_PHOTOS = 5
const MIN_TITLE_LENGTH = 3
const MAX_TITLE_LENGTH = 200
const MAX_DESCRIPTION_LENGTH = 3000

interface StoredItem extends Item {
  userId: number
}

function delay(ms = 400): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function readItems(): StoredItem[] {
  return storage.get<StoredItem[]>(ITEMS_DB_KEY) ?? []
}

function writeItems(items: StoredItem[]): void {
  storage.set(ITEMS_DB_KEY, items)
}

function toItem(stored: StoredItem): Item {
  return {
    id: stored.id,
    title: stored.title,
    description: stored.description,
    condition: stored.condition,
    category: stored.category,
    price_min: stored.price_min,
    price_max: stored.price_max,
    wanted_category: stored.wanted_category,
    wish: stored.wish,
    photos: stored.photos,
    status: stored.status,
    created_at: stored.created_at,
  }
}

function validate(input: CreateItemInput): void {
  const titleLength = input.title.trim().length
  if (titleLength < MIN_TITLE_LENGTH || titleLength > MAX_TITLE_LENGTH) {
    throw new ApiError('validation_error', 'title must contain between 3 and 200 characters')
  }
  if (input.description.length > MAX_DESCRIPTION_LENGTH) {
    throw new ApiError('validation_error', 'description must not exceed 3000 characters')
  }
  if (!input.category_name.trim()) {
    throw new ApiError('validation_error', 'category_name is required')
  }
  if (!input.wanted_category_name.trim()) {
    throw new ApiError('validation_error', 'wanted_category_name is required')
  }
  if (input.price_min <= 0 || input.price_max <= 0) {
    throw new ApiError('validation_error', 'price_min and price_max must be greater than zero')
  }
  if (input.price_min > input.price_max) {
    throw new ApiError('validation_error', 'price_min must not exceed price_max')
  }
  if (input.photos.length > MAX_PHOTOS) {
    throw new ApiError('too_many_photos', 'too many photos')
  }
}

// два объявления: одно с уже найденными цепочками обмена, второе - ещё в поиске
function ensureSeeded(userId: number): void {
  const items = readItems()
  if (items.some((item) => item.id === mockSeedIds.item1(userId))) return

  const musicCategory = mockCategoryStore.getOrCreate('Музыкальные инструменты')
  const drumsKeysCategory = mockCategoryStore.getOrCreate('Ударные и клавишные')
  const stringsCategory = mockCategoryStore.getOrCreate('Гитары и струнные')
  const now = new Date().toISOString()

  const synth: StoredItem = {
    id: mockSeedIds.item1(userId),
    userId,
    title: 'Синтезатор Yamaha PSR',
    description: '',
    condition: 'good',
    category: musicCategory,
    price_min: 38000,
    price_max: 55000,
    wanted_category: drumsKeysCategory,
    wish: {
      id: mockSeedIds.wish1(userId),
      category: drumsKeysCategory,
      price_min: 38000,
      price_max: 55000,
      title_keywords: '',
    },
    photos: [],
    status: 'searching',
    created_at: now,
  }

  const drums: StoredItem = {
    id: mockSeedIds.item2(userId),
    userId,
    title: 'Барабанная установка',
    description: '',
    condition: 'good',
    category: drumsKeysCategory,
    price_min: 45000,
    price_max: 50000,
    wanted_category: stringsCategory,
    wish: {
      id: mockSeedIds.wish2(userId),
      category: stringsCategory,
      price_min: 45000,
      price_max: 50000,
      title_keywords: '',
    },
    photos: [],
    status: 'searching',
    created_at: now,
  }

  writeItems([...items, synth, drums])
}

export const itemApi = {
  async listMine(userId: number): Promise<Item[]> {
    await delay()
    ensureSeeded(userId)

    return readItems()
      .filter((item) => item.userId === userId)
      .sort((a, b) => b.created_at.localeCompare(a.created_at))
      .map(toItem)
  },

  async getById(userId: number, itemId: number): Promise<Item> {
    await delay()
    ensureSeeded(userId)

    const found = readItems().find((item) => item.id === itemId)
    if (!found) {
      throw new ApiError('item_not_found', 'item not found')
    }

    return toItem(found)
  },

  async create(userId: number, input: CreateItemInput): Promise<Item> {
    await delay()

    validate(input)

    const items = readItems()
    const category = mockCategoryStore.getOrCreate(input.category_name.trim())
    const wantedCategory = mockCategoryStore.getOrCreate(input.wanted_category_name.trim())
    const created: StoredItem = {
      id: items.reduce((maxId, item) => Math.max(maxId, item.id), 0) + 1,
      userId,
      title: input.title.trim(),
      description: input.description.trim(),
      condition: input.condition,
      category,
      price_min: input.price_min,
      price_max: input.price_max,
      wanted_category: wantedCategory,
      wish: {
        id: Date.now(),
        category: wantedCategory,
        price_min: input.price_min,
        price_max: input.price_max,
        title_keywords: '',
      },
      photos: input.photos,
      status: 'searching',
      created_at: new Date().toISOString(),
    }
    writeItems([...items, created])

    return toItem(created)
  },

  async uploadPhoto(file: File): Promise<PhotoResponse> {
    await delay(600)

    return {
      id: Date.now() + Math.round(Math.random() * 1000),
      url: URL.createObjectURL(file),
    }
  },
}
