import { ConfigProvider } from 'antd'
import type { PropsWithChildren } from 'react'

const theme = {
  token: {
    fontFamily: "'Manrope', -apple-system, sans-serif",
    colorPrimary: '#01abff',
    colorPrimaryHover: '#0092d9',
  },
  components: {
    Button: {
      primaryColor: '#ffffff',
      defaultBg: '#f2f1f0',
      defaultColor: '#000000',
      defaultBorderColor: 'transparent',
      defaultHoverBg: '#e4e3e2',
      defaultHoverColor: '#000000',
      defaultHoverBorderColor: 'transparent',
      defaultActiveBg: '#e4e3e2',
      defaultActiveColor: '#000000',
      defaultActiveBorderColor: 'transparent',
    },
  },
}

export const AntdProvider = ({ children }: PropsWithChildren) => {
  return <ConfigProvider theme={theme}>{children}</ConfigProvider>
}
