import { CreateItemForm } from '@features/item/create-item'
import styles from './AddItemPage.module.scss'

export const AddItemPage = () => {
  return (
    <div className={styles.page}>
      <CreateItemForm />
    </div>
  )
}
