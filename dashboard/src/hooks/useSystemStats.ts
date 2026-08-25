import { useState, useCallback } from 'react'
import api from '../api'
import { Stats, SystemMetrics } from '../types'

const useSystemStats = (token: string) => {
  const [stats, setStats] = useState<Stats>({ total_tasks: 0, total_bandwidth: 0, users_count: 0 })
  const [system, setSystem] = useState<SystemMetrics>({ cpu: 0, ram: 0, disk: 0, uptime: 0, os: '', arch: '' })

  const fetchStats = useCallback(async () => {
    if (!token) return
    try {
      const [statsRes, sysRes] = await Promise.all([
        api.get('/api/stats'),
        api.get('/api/system'),
      ])

      if (statsRes.data) setStats(statsRes.data)
      if (sysRes.data) setSystem(sysRes.data)
    } catch (err) {
      console.error(err)
    }
  }, [token])

  return { stats, system, fetchStats, setSystem, setStats }
}

export default useSystemStats
