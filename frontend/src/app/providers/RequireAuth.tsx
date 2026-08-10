import { useEffect, useState, type PropsWithChildren } from 'react'
import { Navigate } from 'react-router-dom'
import { observer } from 'mobx-react-lite'
import { sessionStore } from '@entities/session'
import { authApi } from '@shared/api/authApi'

export const RequireAuth = observer(({ children }: PropsWithChildren) => {
  const [checked, setChecked] = useState(false)

  useEffect(() => {
    if (!sessionStore.accessToken) {
      setChecked(true)
      return
    }

    authApi
      .me()
      .then((user) => sessionStore.setUser(user))
      .catch(() => sessionStore.clear())
      .finally(() => setChecked(true))
  }, [])

  if (!checked) return null
  if (!sessionStore.isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
})
