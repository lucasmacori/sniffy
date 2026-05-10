import { ReactQueryProvider } from '@/components/ReactQueryProvider'

export function Providers({ children }: { children: React.ReactNode }) {
  return <ReactQueryProvider>{children}</ReactQueryProvider>
}
