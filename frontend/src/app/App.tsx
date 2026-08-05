import { Typography } from 'antd'
import { AntdProvider } from '@app/providers/AntdProvider'

export const App = () => {
  return (
    <AntdProvider>
      <Typography.Title level={3}>Frontend initialized</Typography.Title>
    </AntdProvider>
  )
}
