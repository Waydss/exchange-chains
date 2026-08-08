const MONTHS_GENITIVE = [
  'января',
  'февраля',
  'марта',
  'апреля',
  'мая',
  'июня',
  'июля',
  'августа',
  'сентября',
  'октября',
  'ноября',
  'декабря',
]

export const formatJoinDate = (isoDate: string): string => {
  const date = new Date(isoDate)
  return `на платформе с ${MONTHS_GENITIVE[date.getMonth()]} ${date.getFullYear()}`
}
