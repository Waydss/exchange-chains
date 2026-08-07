import { Card, Typography } from 'antd'
import { Link } from 'react-router-dom'
import { RegisterForm } from '@features/auth/register'
import styles from './RegisterPage.module.scss'

export const RegisterPage = () => {
  return (
    <div className={styles.wrapper}>
      <Card className={styles.card}>
        <Typography.Title level={3} className={styles.title}>
          Регистрация
        </Typography.Title>

        <RegisterForm />

        <Typography.Paragraph className={styles.footer}>
          Уже есть аккаунт? <Link to="/login">Войти</Link>
        </Typography.Paragraph>
      </Card>
    </div>
  )
}
