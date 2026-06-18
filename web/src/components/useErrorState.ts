import { useCallback, useRef, useState } from 'react'

type ErrorFeedback = {
  message: string
  eventKey: number
}

export function useErrorState(): [ErrorFeedback, (message: string) => void, () => void] {
  const [error, setError] = useState<ErrorFeedback>({ message: '', eventKey: 0 })
  const eventSeqRef = useRef(0)

  const reportError = useCallback((message: string) => {
    eventSeqRef.current += 1
    setError({ message, eventKey: eventSeqRef.current })
  }, [])

  const clearError = useCallback(() => {
    setError((current) => ({ message: '', eventKey: current.eventKey }))
  }, [])

  return [error, reportError, clearError]
}
