export const formatPrice = (value: number): string => `${value.toLocaleString('ru-RU')} ₽`

export const formatPriceRange = (min: number, max: number): string =>
  min === max ? formatPrice(min) : `${min.toLocaleString('ru-RU')} – ${max.toLocaleString('ru-RU')} ₽`

