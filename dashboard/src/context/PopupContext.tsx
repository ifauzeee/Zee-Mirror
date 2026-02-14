import { createContext } from 'react'

export interface PopupContextType {
    showConfirm: (title: string, content: React.ReactNode, options?: any) => Promise<boolean>
    showAlert: (title: string, content: React.ReactNode, options?: any) => Promise<boolean>
    showToast: (message: string, type?: 'info' | 'success' | 'error' | 'alert', duration?: number) => void
}

export const PopupContext = createContext<PopupContextType | null>(null)
