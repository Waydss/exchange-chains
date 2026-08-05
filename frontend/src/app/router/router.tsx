import { createBrowserRouter } from 'react-router-dom'
import { MainPage } from '@pages/main-page'
import { NotFoundPage } from '@pages/not-found-page'

export const router = createBrowserRouter([
  { path: '/', element: <MainPage /> },
  { path: '*', element: <NotFoundPage /> },
])
