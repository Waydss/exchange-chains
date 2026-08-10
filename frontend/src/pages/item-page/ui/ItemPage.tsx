import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Skeleton } from 'antd'
import { observer } from 'mobx-react-lite'
import { sessionStore } from '@entities/session'
import { STATUS_META, type Item } from '@entities/item'
import { ChainCard, type Chain } from '@entities/chain'
import { itemApi } from '@shared/api/itemApi'
import { chainApi } from '@shared/api/chainApi'
import { wishApi } from '@shared/api/wishApi'
import { storage } from '@shared/lib/storage'
import { formatPriceRange } from '@shared/lib/formatPrice'
import styles from './ItemPage.module.scss'

const SEARCH_STARTED_KEY_PREFIX = 'chain_search_started:'

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

      // У бэка нет прямой связи item -> wish (ни wish_id на item, ни item_id на wish),
      // а желание создаётся вместе с вещью с точно её price_min/price_max и wanted_category —
      // это единственные поля, по которым его можно однозначно найти среди GET /wishes.
      const wishes = await wishApi.listMine()
      if (cancelled) return

      const matchedWish = wishes
        .filter(
          (wish) =>
            wish.category.id === loadedItem.wanted_category.id &&
            wish.price_min === loadedItem.price_min &&
            wish.price_max === loadedItem.price_max,
        )
        .sort((a, b) => b.created_at.localeCompare(a.created_at))[0]

      if (!matchedWish) {
        setChains([])
        return
      }

      const loadedChains = await chainApi.getByWishedItemId(userId, matchedWish.id)
      if (cancelled) return
      setChains(loadedChains)

      if (loadedChains.length === 0) {
        const searchStartedKey = `${SEARCH_STARTED_KEY_PREFIX}${matchedWish.id}`
        if (!storage.get<boolean>(searchStartedKey)) {
          await chainApi.startSearch(userId, matchedWish.id)
          storage.set(searchStartedKey, true)
        }
      }
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

        <div className={styles['wish-card']}>
          <span className={styles['wish-label']}>Хочу взамен</span>
          <span className={styles['wish-category']}>{item.wanted_category.name}</span>
          <span className={styles['wish-price']}>
            {formatPriceRange(item.price_min, item.price_max)}
          </span>
        </div>
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
