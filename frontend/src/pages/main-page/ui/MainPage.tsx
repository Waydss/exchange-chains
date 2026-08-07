import { Button, Space, Typography } from 'antd'
import { observer } from 'mobx-react-lite'
import { sessionStore } from '@entities/session'
import styles from './MainPage.module.scss'

export const MainPage = observer(() => {
  return (
    <div className={styles.page}>
      <Space direction="vertical" size="middle">
        <Typography.Title level={3}>Привет, {sessionStore.user?.name}</Typography.Title>
        <Button onClick={() => sessionStore.clear()}>Выйти</Button>
      </Space>
    </div>
  )
})
