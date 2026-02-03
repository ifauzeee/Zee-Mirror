import React, { createContext, useContext, useState, useCallback } from 'react';
import { AnimatePresence } from 'framer-motion';
import Dialog from '../components/Popups/Dialog';
import Toast from '../components/Popups/Toast';

const PopupContext = createContext(null);

export const PopupProvider = ({ children }) => {
    const [modal, setModal] = useState(null); // { title, content, type, onConfirm, confirmText, cancelText }
    const [toasts, setToasts] = useState([]);

    const showConfirm = useCallback((title, content, options = {}) => {
        return new Promise((resolve) => {
            setModal({
                title,
                content,
                type: 'confirm',
                confirmText: options.confirmText || 'Confirm',
                cancelText: options.cancelText || 'Cancel',
                onConfirm: () => {
                    setModal(null);
                    resolve(true);
                },
                onClose: () => {
                    setModal(null);
                    resolve(false);
                },
                ...options
            });
        });
    }, []);

    const showAlert = useCallback((title, content, options = {}) => {
        return new Promise((resolve) => {
            setModal({
                title,
                content,
                type: options.type || 'info', // Can be error, success, info, alert
                confirmText: options.confirmText || 'OK',
                showCancel: false,
                onConfirm: () => {
                    setModal(null);
                    resolve(true);
                },
                onClose: () => {
                    setModal(null);
                    resolve(true);
                },
                ...options
            });
        });
    }, []);

    const showToast = useCallback((message, type = 'info', duration = 3000) => {
        const id = Date.now();
        setToasts((prev) => [...prev, { id, message, type, duration }]);
    }, []);

    const removeToast = useCallback((id) => {
        setToasts((prev) => prev.filter((t) => t.id !== id));
    }, []);

    return (
        <PopupContext.Provider value={{ showConfirm, showAlert, showToast }}>
            {children}

            {/* Modal Render */}
            <Dialog
                isOpen={!!modal}
                onClose={modal?.onClose}
                onConfirm={modal?.onConfirm}
                title={modal?.title}
                type={modal?.type}
                confirmText={modal?.confirmText}
                cancelText={modal?.cancelText}
                showCancel={modal?.showCancel}
            >
                {modal?.content}
            </Dialog>

            {/* Toasts Render */}
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
    );
};

export const usePopups = () => {
    const context = useContext(PopupContext);
    if (!context) {
        throw new Error('usePopups must be used within a PopupProvider');
    }
    return context;
};
