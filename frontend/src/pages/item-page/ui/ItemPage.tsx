import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Skeleton } from 'antd'
import { observer } from 'mobx-react-lite'
import { sessionStore } from '@entities/session'
import { STATUS_META, type Item } from '@entities/item'
import { ChainCard, type Chain } from '@entities/chain'
import { itemApi } from '@shared/api/itemApi'
import { chainApi } from '@shared/api/chainApi'
import { formatPriceRange } from '@shared/lib/formatPrice'
import styles from './ItemPage.module.scss'

export const ItemPage = observer(() => {
  const { itemId } = useParams<{ itemId: string }>()
  const userId = sessionStore.user?.id
  const [item, setItem] = useState<Item | null>(null)
  const [chains, setChains] = useState<Chain[] | null>(null)

  useEffect(() => {
    if (!userId || !itemId) return

    let cancelled = false
    setItem(null)
    setChains(null)

    itemApi.getById(Number(itemId)).then(async (loadedItem) => {
      if (cancelled) return
      setItem(loadedItem)

      if (!loadedItem.wish) {
        setChains([])
        return
      }

      const loadedChains = await chainApi.getByWishedItemId(userId, loadedItem.wish.id)
      if (cancelled) return
      setChains(loadedChains)
    })

    return () => {
      cancelled = true
    }
  }, [userId, itemId])

  if (!item || !userId) {
    return (
      <div className={styles.page}>
        <div className={styles.hero}>
          <Skeleton.Image active style={{ width: 300, height: 240 }} />
          <div className={styles['hero-info']}>
            <Skeleton active paragraph={{ rows: 2 }} />
          </div>
        </div>
        <div className={styles.grid}>
          {[1, 2].map((placeholder) => (
            <div key={placeholder} className={styles['skeleton-card']}>
              <Skeleton.Image active style={{ width: '100%', height: 190 }} />
              <Skeleton active paragraph={{ rows: 2 }} title={false} />
            </div>
          ))}
        </div>
      </div>
    )
  }

  const statusMeta = STATUS_META[item.status]
  const coverPhoto = item.photos[0]

  return (
    <div className={styles.page}>
      <div className={styles.hero}>
        <div className={styles.photo}>
          {coverPhoto ? (
            <img src={coverPhoto.url} alt={item.title} className={styles.image} />
          ) : (
            <span className={styles['photo-placeholder']}>Нет фото</span>
          )}
        </div>

        <div className={styles['hero-info']}>
          <span className={styles.title}>{item.title}</span>
          <span className={styles.subtitle}>{item.category.name}</span>
          <span
            className={styles.status}
            style={{ background: statusMeta.background, color: statusMeta.color }}
          >
            {statusMeta.label}
          </span>
        </div>

        {item.wish && (
          <div className={styles['wish-card']}>
            <span className={styles['wish-label']}>Хочу взамен</span>
            <span className={styles['wish-category']}>{item.wish.category.name}</span>
            <span className={styles['wish-price']}>
              {formatPriceRange(item.wish.price_min, item.wish.price_max)}
            </span>
          </div>
        )}
      </div>

      <div className={styles.section}>
        <span className={styles['section-title']}>Что можно получить за «{item.title}»</span>

        {chains === null ? (
          <div className={styles.grid}>
            {[1, 2].map((placeholder) => (
              <div key={placeholder} className={styles['skeleton-card']}>
                <Skeleton.Image active style={{ width: '100%', height: 190 }} />
                <Skeleton active paragraph={{ rows: 2 }} title={false} />
              </div>
            ))}
          </div>
        ) : chains.length > 0 ? (
          <div className={styles.grid}>
            {chains.map((chain) => (
              <ChainCard key={chain.id} chain={chain} userId={userId} />
            ))}
          </div>
        ) : (
          <div className={styles.empty}>
            <span className={styles['empty-icon']}>?</span>
            <span className={styles['empty-title']}>Ищем подходящую цепочку обмена</span>
            <span className={styles['empty-text']}>
              Для «{item.title}» пока не нашлось совпадений. Мы продолжаем поиск и уведомим, как
              только цепочка сложится.
            </span>
          </div>
        )}
      </div>
    </div>
  )
})
