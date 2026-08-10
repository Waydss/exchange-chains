import { Button } from 'antd'
import { formatPriceRange } from '@shared/lib/formatPrice'
import { getReceivedBarter } from '../model/chainHelpers'
import type { Chain } from '../model/types'
import styles from './ChainCard.module.scss'

interface ChainCardProps {
  chain: Chain
  userId: number
}

export const ChainCard = ({ chain, userId }: ChainCardProps) => {
  const receivedBarter = getReceivedBarter(chain, userId)
  if (!receivedBarter) return null

  const give = receivedBarter.give

  return (
    <div className={styles.card}>
      <div className={styles.photo}>
        <span className={styles['photo-placeholder']}>Нет фото</span>
      </div>

      <div className={styles.body}>
        <span className={styles.title}>{give.title}</span>
        <span className={styles.subtitle}>{formatPriceRange(give.price_min, give.price_max)}</span>
        <span className={styles.length}>цепочка из {chain.barters.length}</span>
        {/* Детали цепочки и подтверждение участия — следующая таска, пока заглушка */}
        <Button className={styles.button} disabled>
          Посмотреть детали
        </Button>
      </div>
    </div>
  )
}
