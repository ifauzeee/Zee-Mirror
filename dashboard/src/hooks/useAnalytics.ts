import { useState, useCallback } from 'react'
import api from '../api'

interface AnalyticsDataPoint {
  Date: string
  TotalTasks: number
  [key: string]: unknown
}

const useAnalytics = (token: string) => {
  const [analyticsData, setAnalyticsData] = useState<AnalyticsDataPoint[]>([])

  const fetchAnalytics = useCallback(async () => {
    if (!token) return
    try {
      const res = await api.get('/api/analytics')
      setAnalyticsData(res.data || [])
    } catch (err) {
      console.error('Analytics error:', err)
    }
  }, [token])

  return { analyticsData, fetchAnalytics }
}

export default useAnalytics
