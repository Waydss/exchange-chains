import { storage } from '@shared/lib/storage'
import type { CategoryResponse } from '@entities/item'

// хранилище категорий для моков itemApi и chainApi

const CATEGORIES_DB_KEY = 'mock_categories_db'

function readCategories(): CategoryResponse[] {
  return storage.get<CategoryResponse[]>(CATEGORIES_DB_KEY) ?? []
}

function writeCategories(categories: CategoryResponse[]): void {
  storage.set(CATEGORIES_DB_KEY, categories)
}

export const mockCategoryStore = {
  getOrCreate(name: string): CategoryResponse {
    const categories = readCategories()
    const existing = categories.find(
      (category) => category.name.toLowerCase() === name.toLowerCase(),
    )
    if (existing) return existing

    const created: CategoryResponse = {
      id: categories.reduce((maxId, category) => Math.max(maxId, category.id), 0) + 1,
      name,
    }
    writeCategories([...categories, created])
    return created
  },

  getById(id: number): CategoryResponse | undefined {
    return readCategories().find((category) => category.id === id)
  },
}
