import { Avatar, Skeleton } from 'antd'
import type { PublicUser } from '@entities/session'
import { formatJoinDate } from '@shared/lib/formatJoinDate'
import { formatRatingStars } from '@shared/lib/rating'
import styles from './ProfileCard.module.scss'

interface ProfileCardProps {
  user: PublicUser
}

export const ProfileCard = ({ user }: ProfileCardProps) => {
  return (
    <div className={styles.card}>
      <Avatar size={90} className={styles.avatar}>
        {user.name[0]?.toUpperCase()}
      </Avatar>
      <span className={styles.name}>{user.name}</span>
      <span className={styles.since}>{formatJoinDate(user.created_at)}</span>

      <div className={styles.rating}>
        <span className={styles.stars}>{formatRatingStars(user.public_rating)}</span>
        <span className={styles['rating-value']}>{user.public_rating.toFixed(1)}</span>
      </div>
      <span className={styles.since}>публичный рейтинг</span>

      <div className={styles.stats}>
        <div className={styles.stat}>
          <Skeleton.Input active size="small" style={{ width: 32, minWidth: 32 }} />
          <span className={styles['stat-label']}>бартеров</span>
        </div>
        <div className={styles.stat}>
          <Skeleton.Input active size="small" style={{ width: 32, minWidth: 32 }} />
          <span className={styles['stat-label']}>в процессе</span>
        </div>
      </div>
    </div>
  )
}
