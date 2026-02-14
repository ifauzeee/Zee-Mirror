import { useState, useCallback, ReactNode } from 'react'
import { AnimatePresence } from 'framer-motion'
import Dialog from '../components/Popups/Dialog'
import Toast from '../components/Popups/Toast'
import { PopupContext } from './PopupContext'

interface PopupProviderProps {
  children: ReactNode
}

interface ModalState {
  title: string
  content: ReactNode
  type: 'confirm' | 'info' | 'success' | 'error' | 'alert'
  confirmText?: string
  cancelText?: string
  showCancel?: boolean
  onConfirm: () => void
  onClose: () => void
  [key: string]: any
}

interface ToastState {
  id: number
  message: string
  type: 'info' | 'success' | 'error' | 'alert'
  duration: number
}

export const PopupProvider: React.FC<PopupProviderProps> = ({ children }) => {
  const [modal, setModal] = useState<ModalState | null>(null)
  const [toasts, setToasts] = useState<ToastState[]>([])

  const showConfirm = useCallback((title: string, content: ReactNode, options: any = {}) => {
    return new Promise<boolean>((resolve) => {
      setModal({
        title,
        content,
        type: 'confirm',
        confirmText: options.confirmText || 'Confirm',
        cancelText: options.cancelText || 'Cancel',
        showCancel: true,
        onConfirm: () => {
          setModal(null)
          resolve(true)
        },
        onClose: () => {
          setModal(null)
          resolve(false)
        },
        ...options,
      })
    })
  }, [])

  const showAlert = useCallback((title: string, content: ReactNode, options: any = {}) => {
    return new Promise<boolean>((resolve) => {
      setModal({
        title,
        content,
        type: options.type || 'info',
        confirmText: options.confirmText || 'OK',
        showCancel: false,
        onConfirm: () => {
          setModal(null)
          resolve(true)
        },
        onClose: () => {
          setModal(null)
          resolve(true)
        },
        ...options,
      })
    })
  }, [])

  const showToast = useCallback((message: string, type: 'info' | 'success' | 'error' | 'alert' = 'info', duration = 3000) => {
    const id = Date.now()
    setToasts((prev) => [...prev, { id, message, type, duration }])
  }, [])

  const removeToast = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  return (
    <PopupContext.Provider value={{ showConfirm, showAlert, showToast }}>
      {children}

      <Dialog
        isOpen={!!modal}
        onClose={modal?.onClose || (() => { })}
        onConfirm={modal?.onConfirm || (() => { })}
        title={modal?.title || ''}
        type={modal?.type || 'info'}
        confirmText={modal?.confirmText}
        cancelText={modal?.cancelText}
        showCancel={modal?.showCancel}
      >
        {modal?.content}
      </Dialog>

      <div className="fixed bottom-8 right-8 z-[110] flex flex-col items-end space-y-4 pointer-events-none">
        <AnimatePresence>
          {toasts.map((toast) => (
            <Toast
              key={toast.id}
              message={toast.message}
              type={toast.type}
              duration={toast.duration}
              onClose={() => removeToast(toast.id)}
            />
          ))}
        </AnimatePresence>
      </div>
    </PopupContext.Provider>
  )
}
