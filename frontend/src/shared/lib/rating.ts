const STARS_TOTAL = 5

export const formatRatingStars = (rating: number): string => {
  const filled = Math.min(STARS_TOTAL, Math.max(0, Math.round(rating)))
  return '★'.repeat(filled) + '☆'.repeat(STARS_TOTAL - filled)
}
