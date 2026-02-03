import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { X, AlertCircle, HelpCircle, Info, CheckCircle2 } from 'lucide-react';
import { cn } from '../../utils/cn';

const Dialog = ({
    isOpen,
    onClose,
    title,
    children,
    type = 'info',
    confirmText = 'Confirm',
    cancelText = 'Cancel',
    onConfirm,
    showCancel = true,
    loading = false
}) => {
    const icons = {
        info: <Info className="text-blue-500" size={24} />,
        confirm: <HelpCircle className="text-primary" size={24} />,
        alert: <AlertCircle className="text-amber-500" size={24} />,
        error: <AlertCircle className="text-red-500" size={24} />,
        success: <CheckCircle2 className="text-green-500" size={24} />
    };

    return (
        <AnimatePresence>
            {isOpen && (
                <div className="fixed inset-0 z-[100] flex items-center justify-center p-4">
                    <motion.div
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        onClick={onClose}
                        className="absolute inset-0 bg-slate-950/40 backdrop-blur-sm"
                    />

                    <motion.div
                        initial={{ opacity: 0, scale: 0.9, y: 20 }}
                        animate={{ opacity: 1, scale: 1, y: 0 }}
                        exit={{ opacity: 0, scale: 0.9, y: 20 }}
                        className="relative w-full max-w-md overflow-hidden glass-card rounded-[2.5rem] p-8 shadow-2xl border border-white/10"
                    >
                        <div className="flex items-center space-x-4 mb-6">
                            <div className={cn(
                                "p-3 rounded-2xl bg-white dark:bg-zinc-800/50 shadow-inner",
                                type === 'confirm' && "text-primary",
                                type === 'alert' && "text-amber-500",
                                type === 'error' && "text-red-500",
                                type === 'success' && "text-green-500",
                                type === 'info' && "text-blue-500"
                            )}>
                                {icons[type]}
                            </div>
                            <h3 className="text-xl font-black tracking-tight text-slate-900 dark:text-white uppercase">
                                {title}
                            </h3>
                            <button
                                onClick={onClose}
                                className="ml-auto p-2 text-slate-400 hover:text-slate-600 dark:hover:text-white transition-colors"
                            >
                                <X size={20} />
                            </button>
                        </div>

                        <div className="text-slate-500 dark:text-zinc-400 font-medium leading-relaxed mb-8 px-1">
                            {children}
                        </div>

                        <div className="flex items-center space-x-4">
                            {showCancel && (
                                <button
                                    onClick={onClose}
                                    className="flex-1 px-6 py-4 rounded-2xl font-bold text-sm uppercase tracking-widest text-slate-500 dark:text-zinc-400 bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 transition-all border border-transparent hover:border-slate-300 dark:hover:border-white/10"
                                >
                                    {cancelText}
                                </button>
                            )}
                            <button
                                onClick={onConfirm}
                                disabled={loading}
                                className={cn(
                                    "flex-1 px-6 py-4 rounded-2xl font-black text-sm uppercase tracking-[0.2em] text-white transition-all shadow-lg active:scale-95 disabled:opacity-50 disabled:pointer-events-none",
                                    type === 'error' ? "bg-red-500 shadow-red-500/20" : "premium-gradient shadow-primary/20",
                                    !showCancel && "w-full"
                                )}
                            >
                                {loading ? 'Processing...' : confirmText}
                            </button>
                        </div>
                    </motion.div>
                </div>
            )}
        </AnimatePresence>
    );
};

export default Dialog;
