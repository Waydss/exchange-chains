import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Alert, Button, ConfigProvider, Form, Input, InputNumber, Select, Upload } from 'antd'
import type { UploadFile, UploadProps } from 'antd'
import { observer } from 'mobx-react-lite'
import { sessionStore } from '@entities/session'
import { CONDITION_LABELS, type Condition, type PhotoResponse } from '@entities/item'
import { itemApi } from '@shared/api/itemApi'
import { CATEGORY_NAMES } from '@shared/config/categories'
import { formatPriceRange } from '@shared/lib/formatPrice'
import styles from './CreateItemForm.module.scss'

const TOLERANCE_OPTIONS = [5, 10, 15, 20, 25, 30].map((percent) => ({
  value: percent,
  label: `±${percent}%`,
}))

const CONDITION_OPTIONS = (Object.entries(CONDITION_LABELS) as [Condition, string][]).map(
  ([value, label]) => ({ value, label }),
)

const CATEGORY_OPTIONS = CATEGORY_NAMES.map((name) => ({ value: name, label: name }))

const MAX_PHOTOS = 5

interface CreateItemFormValues {
  title: string
  description?: string
  condition: Condition
  categoryName: string
  price: number
  tolerancePercent: number
  wantedCategoryName: string
}

function computePriceRange(price: number, tolerancePercent: number): { min: number; max: number } {
  const min = Math.max(1, Math.round(price * (1 - tolerancePercent / 100)))
  const max = Math.round(price * (1 + tolerancePercent / 100))
  return { min, max }
}

export const CreateItemForm = observer(() => {
  const navigate = useNavigate()
  const [form] = Form.useForm<CreateItemFormValues>()
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [photos, setPhotos] = useState<PhotoResponse[]>([])
  const [fileList, setFileList] = useState<UploadFile[]>([])

  const price = Form.useWatch('price', form)
  const tolerancePercent = Form.useWatch('tolerancePercent', form)

  const priceRangePreview = useMemo(() => {
    if (!price || !tolerancePercent) return null
    const { min, max } = computePriceRange(price, tolerancePercent)
    return formatPriceRange(min, max)
  }, [price, tolerancePercent])

  const handleFinish = async (values: CreateItemFormValues) => {
    const userId = sessionStore.user?.id
    if (!userId) return

    setFormError(null)
    setSubmitting(true)

    try {
      const { min, max } = computePriceRange(values.price, values.tolerancePercent)
      await itemApi.create(userId, {
        title: values.title,
        description: values.description ?? '',
        condition: values.condition,
        category_name: values.categoryName,
        price_min: min,
        price_max: max,
        wanted_category_name: values.wantedCategoryName,
        photos,
      })
      navigate('/', { replace: true })
    } catch {
      setFormError('Не удалось опубликовать объявление. Попробуйте ещё раз')
    } finally {
      setSubmitting(false)
    }
  }

  const uploadProps: UploadProps = {
    listType: 'picture-card',
    fileList,
    accept: 'image/jpeg,image/png,image/webp',
    maxCount: MAX_PHOTOS,
    multiple: true,
    customRequest: async ({ file, onSuccess, onError }) => {
      try {
        const uploaded = await itemApi.uploadPhoto(file as File)
        setPhotos((previous) => [...previous, uploaded])
        onSuccess?.(uploaded)
      } catch (error) {
        onError?.(error as Error)
      }
    },
    onChange: ({ fileList: nextFileList }) => setFileList(nextFileList),
    onRemove: (file) => {
      const response = file.response as PhotoResponse | undefined
      if (response) {
        setPhotos((previous) => previous.filter((photo) => photo.id !== response.id))
      }
    },
  }

  return (
    <ConfigProvider
      componentSize="large"
      theme={{
        token: { colorBgContainer: '#f2f1ef' },
        components: {
          Form: {
            labelFontSize: 14,
          },
        },
      }}
    >
      <Form<CreateItemFormValues>
        form={form}
        layout="vertical"
        requiredMark={false}
        onFinish={handleFinish}
        initialValues={{ tolerancePercent: 15 }}
        className={styles.form}
      >
        <span className={styles.heading}>Новое объявление</span>

        {formError && (
          <Form.Item className={styles['alert-item']}>
            <Alert type="error" message={formError} showIcon />
          </Form.Item>
        )}

        <div className={styles.columns}>
          <div className={styles.column}>
            <Form.Item
              className={styles['form-item']}
              name="title"
              label="Название"
              rules={[
                { required: true, message: 'Введите название' },
                { min: 3, max: 200, message: 'Название должно быть от 3 до 200 символов' },
              ]}
            >
              <Input className={styles.input} placeholder="Электрогитара Fender" />
            </Form.Item>

            <Form.Item
              className={styles['form-item']}
              name="description"
              label="Описание"
              rules={[{ max: 3000, message: 'Описание не должно превышать 3000 символов' }]}
            >
              <Input.TextArea
                className={styles.input}
                placeholder="Расскажите об особенностях и состоянии"
                rows={3}
              />
            </Form.Item>

            <div className={styles.row}>
              <Form.Item
                className={styles['form-item']}
                name="categoryName"
                label="Категория"
                rules={[{ required: true, message: 'Укажите категорию' }]}
              >
                <Select
                  className={styles.input}
                  placeholder="Выберите категорию"
                  options={CATEGORY_OPTIONS}
                />
              </Form.Item>

              <Form.Item
                className={styles['form-item']}
                name="condition"
                label="Состояние"
                rules={[{ required: true, message: 'Выберите состояние' }]}
              >
                <Select
                  className={styles.input}
                  placeholder="Выберите состояние"
                  options={CONDITION_OPTIONS}
                />
              </Form.Item>
            </div>

            <Form.Item
              className={styles['form-item']}
              name="wantedCategoryName"
              label="Что хочу взамен"
              rules={[{ required: true, message: 'Укажите желаемую категорию' }]}
            >
              <Select
                className={styles.input}
                placeholder="Выберите категорию"
                options={CATEGORY_OPTIONS}
              />
            </Form.Item>

            <Form.Item className={styles['form-item']} label="Фото объявления">
              <Upload.Dragger {...uploadProps} className={styles.dragger}>
                <span className={styles['dragger-text']}>Перетащите фото или выберите файл</span>
                <span className={styles['dragger-hint']}>
                  JPEG, PNG или WEBP, до {MAX_PHOTOS} фото
                </span>
              </Upload.Dragger>
            </Form.Item>
          </div>

          <div className={styles.column}>
            <div className={styles.row}>
              <Form.Item
                className={styles['form-item']}
                name="price"
                label="Цена"
                rules={[{ required: true, message: 'Укажите цену' }]}
              >
                <InputNumber
                  className={styles.input}
                  min={1}
                  step={1000}
                  placeholder="45 000"
                  addonAfter="₽"
                />
              </Form.Item>

              <Form.Item
                className={styles['form-item']}
                name="tolerancePercent"
                label="Допустимая разница в цене"
                rules={[{ required: true, message: 'Выберите разницу' }]}
              >
                <Select className={styles.input} options={TOLERANCE_OPTIONS} />
              </Form.Item>
            </div>

            <div className={styles['range-preview']}>
              <span className={styles['range-preview-label']}>Диапазон стоимости</span>
              <span className={styles['range-preview-value']}>{priceRangePreview ?? '—'}</span>
            </div>

            <div className={styles.hint}>
              Подсказка: чем шире диапазон цены и допустимая разница — тем больше шанс, что система
              найдёт подходящую цепочку обмена
            </div>

            <div className={styles.row}>
              <Form.Item className={styles['submit-item']}>
                <Button className={styles['submit-button']} htmlType="submit" loading={submitting}>
                  Опубликовать
                </Button>
              </Form.Item>
            </div>
          </div>
        </div>
      </Form>
    </ConfigProvider>
  )
})
