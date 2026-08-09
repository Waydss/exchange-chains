import { useEffect, useState } from 'react'
import { Button, Skeleton } from 'antd'
import { Link } from 'react-router-dom'
import { observer } from 'mobx-react-lite'
import { sessionStore } from '@entities/session'
import { ItemCard, type Item } from '@entities/item'
import { itemApi } from '@shared/api/itemApi'
import styles from './MainPage.module.scss'

export const MainPage = observer(() => {
  const userId = sessionStore.user?.id
  const [items, setItems] = useState<Item[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!userId) return

    let cancelled = false
    setLoading(true)
    itemApi.listMine(userId).then((result) => {
      if (!cancelled) {
        setItems(result)
        setLoading(false)
      }
    })

    return () => {
      cancelled = true
    }
  }, [userId])

  return (
    <div className={styles.page}>
      <div className={styles.header}>
        <span className={styles.title}>Мои вещи</span>
        <Link to="/item/new">
          <Button className={styles['add-button']}>+ Добавить вещь</Button>
        </Link>
      </div>

      {loading ? (
        <div className={styles.grid}>
          {[1, 2, 3, 4].map((placeholder) => (
            <div key={placeholder} className={styles['skeleton-card']}>
              <Skeleton.Image active style={{ width: '100%', height: 200 }} />
              <Skeleton active paragraph={{ rows: 2 }} title={false} />
            </div>
          ))}
        </div>
      ) : items.length === 0 ? (
        <div className={styles.empty}>
          <span className={styles['empty-title']}>Пока нет ни одной вещи</span>
          <span className={styles['empty-text']}>Добавьте вещь, чтобы начать поиск обмена</span>
          <Link to="/item/new">
            <Button className={styles['add-button']}>+ Добавить вещь</Button>
          </Link>
        </div>
      ) : (
        <div className={styles.grid}>
          {items.map((item) => (
            <Link key={item.id} to={`/item/${item.id}`} className={styles['card-link']}>
              <ItemCard item={item} />
            </Link>
          ))}
        </div>
      )}
    </div>
  )
})
