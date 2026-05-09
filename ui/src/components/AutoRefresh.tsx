'use client'

import { useEffect, useRef, useState } from 'react'
import { RefreshCw } from 'lucide-react'

interface AutoRefreshProps {
  interval?: number
  onRefresh: () => void
  isRefreshing?: boolean
}

export function AutoRefresh({ interval = 10000, onRefresh, isRefreshing = false }: AutoRefreshProps) {
  const [secondsLeft, setSecondsLeft] = useState(interval / 1000)
  const onRefreshRef = useRef(onRefresh)

  // Keep ref in sync without triggering effect re-runs
  useEffect(() => {
    onRefreshRef.current = onRefresh
  }, [onRefresh])

  useEffect(() => {
    const countdown = setInterval(() => {
      setSecondsLeft((prev) => {
        if (prev <= 1) {
          // Schedule refresh after current render cycle to avoid setState-during-render error
          setTimeout(() => {
            onRefreshRef.current()
          }, 0)
          return interval / 1000
        }
        return prev - 1
      })
    }, 1000)

    return () => clearInterval(countdown)
  }, [interval])

  return (
    <div className="flex items-center gap-2 text-sm text-gray-500">
      <RefreshCw
        size={16}
        className={`${isRefreshing ? 'animate-spin' : ''}`}
      />
      <span>Refreshing in {secondsLeft}s</span>
    </div>
  )
}
