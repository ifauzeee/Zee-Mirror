import { useState, useCallback } from 'react'
import axios from 'axios'

interface AnalyticsDataPoint {
  Date: string
  TotalTasks: number
  [key: string]: any
}

const useAnalytics = (token: string) => {
  const [analyticsData, setAnalyticsData] = useState<AnalyticsDataPoint[]>([])

  const fetchAnalytics = useCallback(async () => {
    if (!token) return
    try {
      const config = { headers: { 'X-API-Key': token } }
      const res = await axios.get('/api/analytics', config)
      setAnalyticsData(res.data || [])
    } catch (err) {
      console.error('Analytics error:', err)
    }
  }, [token])

  return { analyticsData, fetchAnalytics }
}

export default useAnalytics
