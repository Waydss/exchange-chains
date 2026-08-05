import { RouterProvider } from 'react-router-dom'
import { AntdProvider } from '@app/providers/AntdProvider'
import { router } from '@app/router/router'

export const App = () => {
  return (
    <AntdProvider>
      <RouterProvider router={router} />
    </AntdProvider>
  )
}
