import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Alert, Button, Form, Input } from 'antd'
import { authApi } from '@shared/api/authApi'
import { ApiError } from '@shared/api/errors'
import { sessionStore } from '@entities/session'
import styles from './LoginForm.module.scss'

interface LoginFormValues {
  login: string
  password: string
}

export const LoginForm = () => {
  const navigate = useNavigate()
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const handleFinish = async (values: LoginFormValues) => {
    setFormError(null)
    setSubmitting(true)

    try {
      const result = await authApi.login(values)
      sessionStore.setSession(result)
      navigate('/', { replace: true })
    } catch (error) {
      if (error instanceof ApiError && error.code === 'invalid_credentials') {
        setFormError('Неверный логин или пароль')
      } else {
        setFormError('Не удалось выполнить вход. Попробуйте ещё раз')
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Form<LoginFormValues> layout="vertical" onFinish={handleFinish} requiredMark={false}>
      {formError && (
        <Form.Item className={styles['alert-item']}>
          <Alert type="error" message={formError} showIcon />
        </Form.Item>
      )}

      <Form.Item
        className={styles['form-item']}
        name="login"
        label="Логин"
        rules={[
          { required: true, message: 'Введите логин' },
          { min: 3, max: 32, message: 'Логин должен быть от 3 до 32 символов' },
        ]}
      >
        <Input className={styles.input} autoComplete="username" placeholder="Логин" />
      </Form.Item>

      <Form.Item
        className={styles['form-item']}
        name="password"
        label="Пароль"
        rules={[
          { required: true, message: 'Введите пароль' },
          { min: 8, message: 'Пароль должен быть не короче 8 символов' },
        ]}
      >
        <Input.Password className={styles.input} autoComplete="current-password" placeholder="Пароль" />
      </Form.Item>

      <Form.Item className={styles['submit-item']}>
        <Button className={styles['submit-button']} type="primary" htmlType="submit" loading={submitting} block>
          Войти
        </Button>
      </Form.Item>
    </Form>
  )
}
