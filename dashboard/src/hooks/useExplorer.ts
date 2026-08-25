import { useState, useCallback } from 'react'
import api from '../api'
import { usePopups } from './usePopups'
import { FileItem } from '../types'

const useExplorer = (token: string) => {
  const [explorerPath, setExplorerPath] = useState<string>('')
  const [explorerFiles, setExplorerFiles] = useState<FileItem[]>([])
  const { showConfirm, showAlert, showToast } = usePopups()

  const fetchExplorer = useCallback(
    async (path = '') => {
      if (!token) return
      try {
        const res = await api.get(`/api/explorer/remote?path=${encodeURIComponent(path)}`)

        const normalizedData: FileItem[] = (res.data || []).map((item: Record<string, unknown>) => ({
          name: item.Name,
          displayName: item.Name,
          isDir: item.IsDir,
          size: item.Size,
          time: item.ModTime,
          status: 'cloud',
        }))

        setExplorerFiles(normalizedData)
        setExplorerPath(path)
      } catch (err) {
        console.error('Explorer error:', err)
      }
    },
    [token],
  )

  const getExternalLink = async (name: string) => {
    try {
      const path = explorerPath ? `${explorerPath}/${name}` : name
      const res = await api.get(
        `/api/explorer/remote/link?path=${encodeURIComponent(path)}`,
      )
      if (res.data.link) {
        window.open(res.data.link, '_blank')
      } else {
        showAlert(
          'Link Unavailable',
          'No public distribution link is currently available for this resource.',
          { type: 'alert' },
        )
      }
    } catch {
      showAlert('Extraction Error', 'Failed to generate a secure cloud link for this resource.', {
        type: 'error',
      })
    }
  }

  const deleteFile = async (name: string) => {
    if (
      await showConfirm(
        'Delete Resource',
        `Permanently delete ${name}? This action will wipe the data from the cloud engine.`,
      )
    ) {
      try {
        const path = explorerPath ? `${explorerPath}/${name}` : name
        await api.delete(`/api/explorer/remote?path=${encodeURIComponent(path)}`)
        showToast('Resource deleted successfully', 'success')
        fetchExplorer(explorerPath)
      } catch {
        showAlert(
          'Elimination error',
          'Failed to delete the specified cloud item from the storage matrix.',
          { type: 'error' },
        )
      }
    }
  }

  const uploadFile = async (file: File, path = '') => {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('path', path)

    try {
      await api.post('/api/explorer/upload', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      showToast(`Resource ${file.name} deployed successfully`, 'success')
      fetchExplorer(explorerPath)
      return { success: true }
    } catch (err: unknown) {
      showAlert('Deployment Error', `Failed to upload ${file.name} to the remote matrix.`, {
        type: 'error',
      })
      return { success: false, error: err }
    }
  }

  return {
    explorerPath,
    setExplorerPath,
    explorerFiles,
    fetchExplorer,
    getExternalLink,
    deleteFile,
    uploadFile,
  }
}

export default useExplorer
