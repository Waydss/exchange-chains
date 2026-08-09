import { ApiError } from './errors'
import { storage } from '@shared/lib/storage'
import type { CategoryResponse, CreateItemInput, Item, PhotoResponse } from '@entities/item'

// пока мок, позже интегрирую с бэком (POST /api/v1/items), загрузка фото мок
// листинга вещей (GET /items) на бэке пока нет вообще, сделал моковым.

const ITEMS_DB_KEY = 'mock_items_db'
const CATEGORIES_DB_KEY = 'mock_categories_db'
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

function readCategories(): CategoryResponse[] {
  return storage.get<CategoryResponse[]>(CATEGORIES_DB_KEY) ?? []
}

function writeCategories(categories: CategoryResponse[]): void {
  storage.set(CATEGORIES_DB_KEY, categories)
}

function getOrCreateCategory(name: string): CategoryResponse {
  const categories = readCategories()
  const existing = categories.find((category) => category.name.toLowerCase() === name.toLowerCase())
  if (existing) return existing

  const created: CategoryResponse = {
    id: categories.reduce((maxId, category) => Math.max(maxId, category.id), 0) + 1,
    name,
  }
  writeCategories([...categories, created])
  return created
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

export const itemApi = {
  async listMine(userId: number): Promise<Item[]> {
    await delay()

    return readItems()
      .filter((item) => item.userId === userId)
      .sort((a, b) => b.created_at.localeCompare(a.created_at))
      .map(toItem)
  },

  async create(userId: number, input: CreateItemInput): Promise<Item> {
    await delay()

    validate(input)

    const items = readItems()
    const created: StoredItem = {
      id: items.reduce((maxId, item) => Math.max(maxId, item.id), 0) + 1,
      userId,
      title: input.title.trim(),
      description: input.description.trim(),
      condition: input.condition,
      category: getOrCreateCategory(input.category_name.trim()),
      price_min: input.price_min,
      price_max: input.price_max,
      wanted_category: getOrCreateCategory(input.wanted_category_name.trim()),
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
