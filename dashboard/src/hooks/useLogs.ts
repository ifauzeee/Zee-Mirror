import { useState, useCallback } from 'react'
import api from '../api'

const useLogs = (token: string) => {
  const [logsContent, setLogsContent] = useState<string>('')

  const fetchLogs = useCallback(async () => {
    if (!token) return
    try {
      const res = await api.get('/api/logs')
      setLogsContent(res.data.logs || '')
    } catch (err) {
      console.error('Logs error:', err)
    }
  }, [token])

  return { logsContent, fetchLogs }
}

export default useLogs
