import { createBrowserRouter } from 'react-router-dom'
import { GuestOnly } from '@app/providers/GuestOnly'
import { RequireAuth } from '@app/providers/RequireAuth'
import { MainPage } from '@pages/main-page'
import { NotFoundPage } from '@pages/not-found-page'
import { LoginPage } from '@pages/login-page'
import { RegisterPage } from '@pages/register-page'

export const router = createBrowserRouter([
  {
    path: '/',
    element: (
      <RequireAuth>
        <MainPage />
      </RequireAuth>
    ),
  },
  {
    path: '/login',
    element: (
      <GuestOnly>
        <LoginPage />
      </GuestOnly>
    ),
  },
  {
    path: '/register',
    element: (
      <GuestOnly>
        <RegisterPage />
      </GuestOnly>
    ),
  },
  { path: '*', element: <NotFoundPage /> },
])
