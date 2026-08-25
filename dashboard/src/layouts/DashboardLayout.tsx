import { useState, useEffect } from 'react'
import { RefreshCcw, Sun, Moon } from 'lucide-react'
import { Outlet, useLocation } from 'react-router-dom'
import Sidebar from '../components/Sidebar/Sidebar'


interface DashboardLayoutProps {
    tasksCount: number
    isDarkMode: boolean
    toggleTheme: () => void
    loading: boolean
    onLogout: () => void
}

const DashboardLayout: React.FC<DashboardLayoutProps> = ({
    tasksCount,
    isDarkMode,
    toggleTheme,
    loading,
    onLogout
}) => {
    const [activeTab, setActiveTab] = useState('overview')
    const location = useLocation()
    useEffect(() => {
        const path = location.pathname.substring(1) || 'overview'

        setActiveTab(path.split('/')[0])
    }, [location])





    return (
        <div className="flex h-screen bg-[#f8fafc] dark:bg-[#050505] text-[#0f172a] dark:text-[#f8fafc] overflow-hidden font-sans selection:bg-primary selection:text-white">
            <Sidebar
                tasksCount={tasksCount}
                onLogout={onLogout}
            />

            <main className="flex-1 overflow-y-auto p-12 scroll-smooth scrollbar-hide relative">
                <div className="absolute top-[-100px] right-[-100px] w-[500px] h-[500px] bg-primary/5 rounded-full blur-[120px] pointer-events-none" />
                <div className="absolute bottom-[-100px] left-[-100px] w-[500px] h-[500px] bg-indigo-500/5 rounded-full blur-[120px] pointer-events-none" />

                <header className="flex flex-col md:flex-row justify-between items-start md:items-center mb-16 gap-8 relative z-20">
                    <div className="animate-in fade-in slide-in-from-left-8 duration-700">
                        <div className="flex items-center space-x-4 mb-4">
                            <div className="h-1.5 w-12 bg-primary rounded-full shadow-[0_0_12px_rgba(59,130,246,0.5)]" />
                            <span className="text-[10px] font-black uppercase tracking-[0.4em] text-primary opacity-80">
                                Cloud Infrastructure Matrix
                            </span>
                        </div>
                        <h2 className="text-6xl font-black text-slate-900 dark:text-white tracking-[-0.04em] mb-3 capitalize leading-[0.9]">
                            {activeTab} <span className="text-primary italic">Engine</span>
                        </h2>
                        <p className="text-sm font-bold text-slate-400 max-w-lg leading-relaxed tracking-tight group">
                            Monitoring high-performance distribution clusters & edge computing resources in{' '}
                            <span className="text-slate-600 dark:text-slate-200 transition-colors">
                                real-time globally
                            </span>
                            .
                        </p>
                    </div>

                    <div className="flex items-center space-x-5 animate-in fade-in slide-in-from-right-8 duration-700">
                        <button
                            onClick={toggleTheme}
                            className="p-4 bg-white dark:bg-zinc-900/60 border border-slate-200 dark:border-white/5 rounded-[1.75rem] shadow-xl hover:scale-110 active:scale-95 transition-all text-slate-500 dark:text-yellow-400 hover:shadow-primary/10"
                        >
                            {isDarkMode ? <Sun size={24} /> : <Moon size={24} />}
                        </button>
                        <div className="px-8 py-4 premium-gradient text-white rounded-[1.75rem] font-black text-[10px] uppercase tracking-[0.2em] shadow-[0_12px_32px_-8px_rgba(59,130,246,0.5)] hover:scale-105 active:scale-95 transition-all group flex items-center space-x-4">
                            <RefreshCcw
                                size={18}
                                className={
                                    loading
                                        ? 'animate-spin'
                                        : 'group-hover:rotate-180 transition-transform duration-700'
                                }
                            />
                            <span>{loading ? 'Refreshing...' : 'Cluster Active'}</span>
                        </div>
                    </div>
                </header>

                <Outlet />
            </main>
        </div>
    )
}

export default DashboardLayout
