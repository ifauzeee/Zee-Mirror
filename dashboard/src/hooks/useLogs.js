import { useState, useCallback } from 'react'
import axios from 'axios'

const useLogs = (token) => {
  const [logsContent, setLogsContent] = useState('')

  const fetchLogs = useCallback(async () => {
    if (!token) return
    try {
      const config = { headers: { 'X-API-Key': token } }
      const res = await axios.get('/api/logs', config)
      setLogsContent(res.data.logs || '')
    } catch (err) {
      console.error('Logs error:', err)
    }
  }, [token])

  return { logsContent, fetchLogs }
}

export default useLogs
