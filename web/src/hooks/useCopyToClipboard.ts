import { useCallback, useState } from 'react'

export function useCopyToClipboard(timeout = 1500) {
  const [copied, setCopied] = useState(false)

  const copyToClipboard = useCallback(
    (text: string) => {
      navigator.clipboard.writeText(text)
      setCopied(true)
      setTimeout(() => setCopied(false), timeout)
    },
    [timeout],
  )

  return { copied, copyToClipboard }
}
