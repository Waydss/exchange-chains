import { ConfigProvider } from 'antd'
import type { PropsWithChildren } from 'react'

export const AntdProvider = ({ children }: PropsWithChildren) => {
  return <ConfigProvider>{children}</ConfigProvider>
}
