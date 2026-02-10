import { useState, useEffect, useCallback } from 'react'

export const useUsers = (apiToken) => {
  const [users, setUsers] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const fetchUsers = useCallback(async () => {
    if (!apiToken) return
    setLoading(true)
    try {
      const response = await fetch('/api/users', {
        headers: { 'X-API-Key': apiToken },
      })
      if (response.ok) {
        const data = await response.json()
        setUsers(data || [])
        setError(null)
      } else {
        setError('Failed to fetch users')
      }
    } catch {
      setError('Network error')
    } finally {
      setLoading(false)
    }
  }, [apiToken])

  const updateUser = async (userData) => {
    try {
      const response = await fetch('/api/users/update', {
        method: 'POST',
        headers: {
          'X-API-Key': apiToken,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(userData),
      })
      if (response.ok) {
        await fetchUsers()
        return { success: true }
      }
      return { success: false, error: 'Update failed' }
    } catch {
      return { success: false, error: 'Network error' }
    }
  }

  const deleteUser = async (id) => {
    try {
      const response = await fetch('/api/users/delete', {
        method: 'POST',
        headers: {
          'X-API-Key': apiToken,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ id }),
      })
      if (response.ok) {
        await fetchUsers()
        return { success: true }
      }
      return { success: false, error: 'Delete failed' }
    } catch {
      return { success: false, error: 'Network error' }
    }
  }

  const addUser = async (userData) => {
    try {
      const response = await fetch('/api/users/add', {
        method: 'POST',
        headers: {
          'X-API-Key': apiToken,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(userData),
      })
      if (response.status === 201) {
        await fetchUsers()
        return { success: true }
      }
      const data = await response.json()
      return { success: false, error: data.error || 'Add failed' }
    } catch {
      return { success: false, error: 'Network error' }
    }
  }

  useEffect(() => {
    fetchUsers()
  }, [fetchUsers])

  return { users, loading, error, fetchUsers, updateUser, deleteUser, addUser }
}
