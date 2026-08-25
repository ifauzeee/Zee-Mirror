import { useState, useCallback } from 'react'
import api from '../api'
import { usePopups } from './usePopups'
import { Task } from '../types'

const useTasks = (token: string) => {
  const [tasks, setTasks] = useState<Task[]>([])
  const { showConfirm, showToast } = usePopups()

  const fetchTasks = useCallback(async () => {
    if (!token) return
    try {
      const res = await api.get('/api/tasks')
      setTasks(res.data || [])
    } catch (err) {
      console.error(err)
    }
  }, [token])

  const cancelTask = async (id: string) => {
    if (
      await showConfirm(
        'Cancel Task',
        'Terminate this distribution process? This action cannot be reversed.',
      )
    ) {
      try {
        await api.delete(`/api/tasks?id=${id}`)
        showToast('Task terminated successfully', 'success')
        fetchTasks()
      } catch (err) {
        console.error(err)
        showToast('Failed to terminate task', 'error')
      }
    }
  }

  return { tasks, fetchTasks, cancelTask, setTasks }
}

export default useTasks
