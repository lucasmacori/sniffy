'use client'

import { useState } from 'react'
import { Eye, EyeOff } from 'lucide-react'

interface SecretRevealProps {
  secret: string
  maxPreview?: number
}

export function SecretReveal({ secret, maxPreview = 8 }: SecretRevealProps) {
  const [revealed, setRevealed] = useState(false)

  const preview = secret.slice(0, maxPreview) + '•••'

  return (
    <div className="flex items-center gap-2">
      <code className="rounded bg-gray-100 px-2 py-1 text-sm font-mono text-gray-800">
        {revealed ? secret : preview}
      </code>
      <button
        onClick={() => setRevealed(!revealed)}
        className="rounded p-1 text-gray-500 hover:bg-gray-100 hover:text-gray-700 transition-colors"
        title={revealed ? 'Hide secret' : 'Reveal secret'}
        type="button"
      >
        {revealed ? <EyeOff size={16} /> : <Eye size={16} />}
      </button>
    </div>
  )
}
