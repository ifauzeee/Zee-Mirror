import { useContext } from 'react'
import { PopupContext } from '../context/PopupContext'

export const usePopups = () => {
  const context = useContext(PopupContext)
  if (!context) {
    throw new Error('usePopups must be used within a PopupProvider')
  }
  return context
}
