import type { PropsWithChildren } from 'react'
import { Navigate } from 'react-router-dom'
import { observer } from 'mobx-react-lite'
import { sessionStore } from '@entities/session'

export const RequireAuth = observer(({ children }: PropsWithChildren) => {
  if (!sessionStore.isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
})
