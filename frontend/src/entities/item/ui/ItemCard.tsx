import { formatPriceRange } from '@shared/lib/formatPrice'
import { STATUS_META } from '../model/statusMeta'
import type { Item } from '../model/types'
import styles from './ItemCard.module.scss'

interface ItemCardProps {
  item: Item
}

export const ItemCard = ({ item }: ItemCardProps) => {
  const statusMeta = STATUS_META[item.status]
  const coverPhoto = item.photos[0]

  return (
    <div className={styles.card}>
      <div className={styles.photo}>
        {coverPhoto ? (
          <img src={coverPhoto.url} alt={item.title} className={styles.image} />
        ) : (
          <span className={styles['photo-placeholder']}>Нет фото</span>
        )}
      </div>

      <div className={styles.body}>
        <span className={styles.title}>{item.title}</span>
        <span className={styles.subtitle}>
          {item.category.name} · {formatPriceRange(item.price_min, item.price_max)}
        </span>
        <span
          className={styles.status}
          style={{ background: statusMeta.background, color: statusMeta.color }}
        >
          {statusMeta.label}
        </span>
      </div>
    </div>
  )
}
