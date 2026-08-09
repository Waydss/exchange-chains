import { Avatar, Dropdown } from 'antd'
import type { MenuProps } from 'antd'
import { observer } from 'mobx-react-lite'
import { NavLink, useNavigate } from 'react-router-dom'
import { sessionStore } from '@entities/session'
import styles from './Header.module.scss'

const navLinkClassName = ({ isActive }: { isActive: boolean }) =>
  isActive ? `${styles.link} ${styles['link-active']}` : styles.link

export const Header = observer(() => {
  const navigate = useNavigate()

  const menuItems: MenuProps['items'] = [
    { key: 'profile', label: 'Профиль' },
    { key: 'logout', label: 'Выйти' },
  ]

  const handleMenuClick: MenuProps['onClick'] = ({ key }) => {
    if (key === 'profile') {
      navigate('/profile')
    } else if (key === 'logout') {
      sessionStore.clear()
      navigate('/login', { replace: true })
    }
  }

  const userInitial = sessionStore.user?.name?.[0]?.toUpperCase()

  return (
    <header className={styles.header}>
      <NavLink to="/" className={styles.logo}>
        <img src="/logo.svg" alt="Trade" className={styles['logo-image']} />
      </NavLink>

      <nav className={styles.nav}>
        <NavLink to="/" end className={navLinkClassName}>
          Мои вещи
        </NavLink>
        <span className={`${styles.link} ${styles['link-disabled']}`} aria-disabled="true">
          История
        </span>
        <NavLink to="/profile" className={navLinkClassName}>
          Профиль
        </NavLink>
      </nav>

      <div className={styles.spacer} />

      <Dropdown menu={{ items: menuItems, onClick: handleMenuClick }} trigger={['click']} placement="bottomRight">
        <Avatar className={styles.avatar}>{userInitial}</Avatar>
      </Dropdown>
    </header>
  )
})
