import { Skeleton } from 'antd'
import styles from './ActiveExchanges.module.scss'

const PLACEHOLDER_ITEMS = [1, 2]

export const ActiveExchanges = () => {
  return (
    <div className={styles.wrapper}>
      <span className={styles.title}>Активные обмены</span>

      {PLACEHOLDER_ITEMS.map((item) => (
        <div key={item} className={styles.card}>
          <Skeleton.Input active size="small" style={{ width: 110, minWidth: 110 }} />
          <Skeleton active avatar paragraph={{ rows: 2 }} title={false} />
        </div>
      ))}
    </div>
  )
}
