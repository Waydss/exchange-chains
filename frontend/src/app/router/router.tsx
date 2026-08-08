import { createBrowserRouter } from 'react-router-dom'
import { GuestOnly } from '@app/providers/GuestOnly'
import { RequireAuth } from '@app/providers/RequireAuth'
import { AppLayout } from '@app/layouts/AppLayout'
import { MainPage } from '@pages/main-page'
import { AddItemPage } from '@pages/add-item-page'
import { ProfilePage } from '@pages/profile-page'
import { NotFoundPage } from '@pages/not-found-page'
import { LoginPage } from '@pages/login-page'
import { RegisterPage } from '@pages/register-page'

export const router = createBrowserRouter([
  {
    path: '/',
    element: (
      <RequireAuth>
        <AppLayout />
      </RequireAuth>
    ),
    children: [
      { index: true, element: <MainPage /> },
      { path: 'item/new', element: <AddItemPage /> },
      { path: 'profile', element: <ProfilePage /> },
    ],
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
