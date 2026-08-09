import { observer } from 'mobx-react-lite'
import { sessionStore } from '@entities/session'
import { ProfileCard } from './ProfileCard'
import { ActiveExchanges } from './ActiveExchanges'
import styles from './ProfilePage.module.scss'

export const ProfilePage = observer(() => {
  const user = sessionStore.user

  if (!user) return null

  return (
    <div className={styles.page}>
      <ProfileCard user={user} />
      <ActiveExchanges />
    </div>
  )
})
