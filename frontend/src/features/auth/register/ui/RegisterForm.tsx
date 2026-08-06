import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Alert, Button, Form, Input } from 'antd'
import { authApi } from '@shared/api/authApi'
import { ApiError } from '@shared/api/errors'
import { sessionStore } from '@entities/session'
import styles from './RegisterForm.module.scss'

interface RegisterFormValues {
  login: string
  name: string
  password: string
  confirmPassword: string
}

const LOGIN_PATTERN = /^[a-zA-Z0-9._-]+$/

export const RegisterForm = () => {
  const navigate = useNavigate()
  const [form] = Form.useForm<RegisterFormValues>()
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const handleFinish = async (values: RegisterFormValues) => {
    setFormError(null)
    setSubmitting(true)

    try {
      const result = await authApi.register({
        login: values.login,
        name: values.name,
        password: values.password,
      })
      sessionStore.setSession(result)
      navigate('/', { replace: true })
    } catch (error) {
      if (error instanceof ApiError && error.code === 'login_already_exists') {
        form.setFields([{ name: 'login', errors: ['Такой логин уже занят'] }])
      } else {
        setFormError('Не удалось зарегистрироваться. Попробуйте ещё раз')
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Form<RegisterFormValues> form={form} layout="vertical" onFinish={handleFinish} requiredMark={false}>
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
          { pattern: LOGIN_PATTERN, message: 'Допустимы латинские буквы, цифры, точка, дефис и подчёркивание' },
        ]}
      >
        <Input className={styles.input} autoComplete="username" placeholder="Логин" />
      </Form.Item>

      <Form.Item
        className={styles['form-item']}
        name="name"
        label="Имя"
        rules={[
          { required: true, message: 'Введите имя' },
          { min: 2, max: 100, message: 'Имя должно быть от 2 до 100 символов' },
        ]}
      >
        <Input className={styles.input} autoComplete="name" placeholder="Имя" />
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
        <Input.Password className={styles.input} autoComplete="new-password" placeholder="Пароль" />
      </Form.Item>

      <Form.Item
        className={styles['form-item']}
        name="confirmPassword"
        label="Повторите пароль"
        dependencies={['password']}
        rules={[
          { required: true, message: 'Повторите пароль' },
          ({ getFieldValue }) => ({
            validator(_, value) {
              if (!value || getFieldValue('password') === value) {
                return Promise.resolve()
              }
              return Promise.reject(new Error('Пароли не совпадают'))
            },
          }),
        ]}
      >
        <Input.Password className={styles.input} autoComplete="new-password" placeholder="Повторите пароль" />
      </Form.Item>

      <Form.Item className={styles['submit-item']}>
        <Button className={styles['submit-button']} type="primary" htmlType="submit" loading={submitting} block>
          Зарегистрироваться
        </Button>
      </Form.Item>
    </Form>
  )
}
