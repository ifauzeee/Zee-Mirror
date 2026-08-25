import {
  Activity,
  Settings,
  Folder,
  Download,
  BarChart3,
  Terminal,
  Bot,
  LogOut,
  GitBranch,
  Users,
  Clock,
} from 'lucide-react'

interface SidebarProps {
  tasksCount: number
  onLogout: () => void
}

import { useNavigate, useLocation } from 'react-router-dom'

const Sidebar: React.FC<SidebarProps> = ({ tasksCount, onLogout }) => {
  const navigate = useNavigate()
  const location = useLocation()
  const currentPath = location.pathname.substring(1) || 'overview'
  const isActiveItem = (itemId: string) => {
    if (itemId === currentPath) return true
    if (currentPath.startsWith(itemId + '/')) return true
    return false
  }
  const navItems = [
    { id: 'overview', label: 'Overview', icon: Activity },
    { id: 'tasks', label: 'Queued Tasks', icon: Download },
    { id: 'tasks/history', label: 'Task History', icon: Clock },
    { id: 'files', label: 'File Explorer', icon: Folder },
    { id: 'analytics', label: 'Analytics', icon: BarChart3 },
    { id: 'logs', label: 'System Logs', icon: Terminal },
    { id: 'users', label: 'User Management', icon: Users },
    { id: 'settings', label: 'Environment', icon: Settings },
  ]

  return (
    <aside className="w-80 bg-white/50 dark:bg-zinc-950/20 backdrop-blur-3xl border-r border-slate-200/50 dark:border-white/5 flex flex-col z-30 shadow-[4px_0_24px_rgba(0,0,0,0.02)] relative h-full">
      <div className="absolute top-0 left-0 w-full h-[500px] bg-gradient-to-b from-primary/5 to-transparent pointer-events-none" />

      <div className="p-8 pb-0 shrink-0 relative z-10">
        <div className="flex items-center space-x-4 px-2 group">
          <div className="p-3.5 premium-gradient rounded-2xl text-white shadow-[0_8px_20px_rgba(59,130,246,0.3)] ring-4 ring-primary/10 group-hover:scale-110 group-hover:rotate-6 transition-all duration-500 cursor-pointer">
            <Bot size={28} className="animate-pulse-soft" />
          </div>
          <div>
            <h1 className="text-2xl font-[900] text-slate-900 dark:text-white tracking-[-0.05em] uppercase leading-none">
              Zee<span className="text-primary italic">Mirror</span>
            </h1>
            <p className="text-[9px] font-black text-primary/60 tracking-[0.4em] uppercase mt-1.5 opacity-80">
              Cloud Interface
            </p>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-8 py-10 space-y-2 relative z-10 scrollbar-hide">
        <nav className="space-y-2">
          {navItems.map((item) => (
            <button
              key={item.id}
              onClick={() => navigate(item.id === 'overview' ? '/' : `/${item.id}`)}
              className={`flex items-center space-x-4 w-full px-6 py-4 rounded-[1.75rem] transition-all duration-500 font-black text-xs uppercase tracking-wider group ${isActiveItem(item.id)
                ? 'bg-primary text-white shadow-[0_12px_24px_-8px_rgba(59,130,246,0.4)] scale-[1.03]'
                : 'text-slate-500 dark:text-zinc-500 hover:bg-slate-100 dark:hover:bg-white/5 hover:text-primary dark:hover:text-primary transition-all'
                }`}
            >
              <item.icon
                size={18}
                className={`${isActiveItem(item.id) ? 'scale-110' : 'group-hover:scale-110 group-hover:text-primary'} transition-all`}
              />
              <span>{item.label}</span>
              {item.id === 'tasks' && tasksCount > 0 && (
                <span
                  className={`ml-auto px-2 py-0.5 rounded-lg text-[9px] font-black ${isActiveItem(item.id) ? 'bg-white text-primary' : 'bg-primary/10 text-primary animate-pulse'}`}
                >
                  {tasksCount}
                </span>
              )}
            </button>
          ))}
        </nav>
      </div>

      <div className="p-8 pt-0 shrink-0 relative z-10 space-y-4 bg-gradient-to-t from-[#f8fafc] dark:from-[#050505] to-transparent via-[#f8fafc]/80 dark:via-[#050505]/80">
        <a
          href="https://github.com/ifauzeee/Zee-Mirror"
          target="_blank"
          rel="noopener noreferrer"
          className="p-5 bg-slate-50/50 dark:bg-zinc-900/40 rounded-[2rem] flex items-center space-x-4 border border-slate-100 dark:border-white/5 shadow-inner hover:bg-slate-100 dark:hover:bg-zinc-900 transition-all group/credit cursor-pointer"
        >
          <div className="w-10 h-10 rounded-xl bg-slate-200 dark:bg-white/10 flex items-center justify-center text-slate-600 dark:text-white shadow-lg group-hover/credit:scale-110 transition-transform">
            <GitBranch size={20} />
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-[9px] font-black uppercase tracking-widest text-slate-500 dark:text-slate-500">
              Maintained By
            </p>
            <p className="text-sm font-black text-slate-800 dark:text-white group-hover/credit:text-primary transition-colors">
              ifauzee
            </p>
          </div>
        </a>

        <button
          onClick={onLogout}
          className="w-full flex items-center justify-center space-x-3 p-5 rounded-[2rem] bg-red-500/5 hover:bg-red-500 text-red-500 hover:text-white border border-red-500/10 transition-all duration-500 group"
        >
          <LogOut size={18} className="group-hover:-translate-x-1 transition-transform" />
          <span className="text-[10px] font-black uppercase tracking-[0.2em]">Exit Engine</span>
        </button>
      </div>
    </aside>
  )
}

export default Sidebar
