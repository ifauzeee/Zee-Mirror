import { useEffect } from 'react'
import { motion } from 'framer-motion'
import { X, CheckCircle2, AlertCircle, Info } from 'lucide-react'
import { cn } from '../../utils/cn'

const Toast = ({ message, type = 'info', onClose, duration = 3000 }) => {
  useEffect(() => {
    const timer = setTimeout(() => {
      onClose()
    }, duration)
    return () => clearTimeout(timer)
  }, [onClose, duration])

  const icons = {
    info: <Info size={18} className="text-blue-500" />,
    success: <CheckCircle2 size={18} className="text-green-500" />,
    error: <AlertCircle size={18} className="text-red-500" />,
    alert: <AlertCircle size={18} className="text-amber-500" />,
  }

  return (
    <motion.div
      initial={{ opacity: 0, x: 20, scale: 0.9 }}
      animate={{ opacity: 1, x: 0, scale: 1 }}
      exit={{ opacity: 0, x: 20, scale: 0.9 }}
      className="flex items-center space-x-4 p-4 pr-3 glass-card rounded-2xl shadow-2xl border border-white/10 max-w-sm pointer-events-auto"
    >
      <div
        className={cn(
          'p-2 rounded-xl bg-white dark:bg-zinc-800/50 shadow-inner shrink-0',
          type === 'success' && 'text-green-500',
          type === 'error' && 'text-red-500',
          type === 'alert' && 'text-amber-500',
          type === 'info' && 'text-blue-500',
        )}
      >
        {icons[type]}
      </div>
      <p className="text-sm font-bold text-slate-700 dark:text-zinc-200 line-clamp-2">{message}</p>
      <button
        onClick={onClose}
        className="p-1 text-slate-400 hover:text-slate-600 dark:hover:text-white transition-colors"
      >
        <X size={16} />
      </button>
    </motion.div>
  )
}

export default Toast
