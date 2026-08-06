import { Card, Typography } from 'antd'
import { Link } from 'react-router-dom'
import { LoginForm } from '@features/auth/login'
import styles from './LoginPage.module.scss'

export const LoginPage = () => {
  return (
    <div className={styles.wrapper}>
      <Card className={styles.card}>
        <Typography.Title level={3} className={styles.title}>
          Вход
        </Typography.Title>

        <LoginForm />

        <Typography.Paragraph className={styles.footer}>
          Нет аккаунта? <Link to="/register">Зарегистрироваться</Link>
        </Typography.Paragraph>
      </Card>
    </div>
  )
}
