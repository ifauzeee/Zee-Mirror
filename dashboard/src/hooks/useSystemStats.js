import { useState, useCallback } from 'react'
import axios from 'axios'

const useSystemStats = (token) => {
  const [stats, setStats] = useState({ total_tasks: 0, total_bandwidth: 0, users_count: 0 })
  const [system, setSystem] = useState({ cpu: 0, ram: 0, disk: 0, uptime: 0, os: '', arch: '' })

  const fetchStats = useCallback(async () => {
    if (!token) return
    try {
      const config = { headers: { 'X-API-Key': token } }
      const [statsRes, sysRes] = await Promise.all([
        axios.get('/api/stats', config),
        axios.get('/api/system', config),
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
