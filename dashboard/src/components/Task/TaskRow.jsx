import { Download, Trash2, TrendingUp, Clock } from 'lucide-react'

const TaskRow = ({ task, formatBytes, onCancel }) => (
  <div className="glass-card p-8 rounded-[3rem] flex flex-col space-y-7 hover:scale-[1.02] transition-all duration-500 border-l-[6px] border-l-primary group relative overflow-hidden shadow-xl hover:shadow-primary/5 text-slate-800 dark:text-white">
    <div className="absolute top-0 right-0 p-6 opacity-[0.02] group-hover:opacity-[0.05] transition-all duration-1000 rotate-12 -mr-8 -mt-8">
      <Download size={120} />
    </div>

    <div className="flex items-start justify-between relative z-10 gap-6">
      <div className="flex items-center space-x-6 flex-1 min-w-0">
        <div className="p-4 bg-primary/10 text-primary rounded-[1.5rem] group-hover:premium-gradient group-hover:text-white transition-all duration-700 shadow-inner shrink-0 group-hover:rotate-3">
          <Download size={28} className={task.status === 'downloading' ? 'animate-bounce' : ''} />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center flex-wrap gap-2 mb-2">
            <span className="px-2.5 py-1 bg-primary text-white text-[9px] font-black uppercase tracking-widest rounded-lg shadow-lg shadow-primary/20">
              {task.type}
            </span>
            <span className="px-2.5 py-1 bg-slate-100 dark:bg-white/5 text-slate-900 dark:text-white text-[8px] font-black uppercase tracking-[0.2em] rounded-lg border border-slate-200 dark:border-white/5">
              {task.status}
            </span>
          </div>
          <h4
            className="font-black text-xl text-slate-900 dark:text-white truncate pr-4 leading-tight group-hover:text-primary transition-colors"
            title={task.fileName || task.id}
          >
            {task.fileName || `Thread: ${task.id}`}
          </h4>
        </div>
      </div>
      <button
        onClick={() => onCancel(task.id)}
        className="p-4 text-slate-400 dark:text-slate-500 hover:text-white hover:bg-red-500 rounded-2xl transition-all shrink-0 active:scale-90 border border-slate-200 dark:border-white/5 hover:border-red-500/20 shadow-sm bg-white dark:bg-transparent"
      >
        <Trash2 size={24} />
      </button>
    </div>

    <div className="space-y-6 relative z-10">
      <div className="flex items-end justify-between px-1">
        <div className="flex items-center space-x-12">
          <div className="space-y-1">
            <p className="text-[10px] font-black text-slate-500 dark:text-slate-400 uppercase tracking-widest leading-none mb-1">
              Bandwidth
            </p>
            <div className="flex items-center text-sm font-black text-slate-900 dark:text-slate-200">
              <TrendingUp size={16} className="mr-2 text-primary" /> {formatBytes(task.speed || 0)}
              <span className="text-[10px] font-bold opacity-70 ml-1">/s</span>
            </div>
          </div>
          <div className="space-y-1">
            <p className="text-[10px] font-black text-slate-500 dark:text-slate-400 uppercase tracking-widest leading-none mb-1">
              Time Buffer
            </p>
            <div className="flex items-center text-sm font-black text-slate-900 dark:text-slate-200">
              <Clock size={16} className="mr-2 text-indigo-400" />{' '}
              {task.eta
                ? `${Math.floor(task.eta / 1e9 / 60)}m ${Math.floor(task.eta / 1e9) % 60}s`
                : 'Stable'}
            </div>
          </div>
        </div>
        <div className="text-right">
          <span className="text-4xl font-black text-primary tracking-tighter tabular-nums drop-shadow-sm">
            {(task.progress || 0).toFixed(1)}
            <span className="text-xs ml-0.5">%</span>
          </span>
        </div>
      </div>

      <div className="relative group/bar">
        <div className="h-2.5 w-full bg-slate-100 dark:bg-white/5 rounded-full overflow-hidden shadow-inner flex p-0.5">
          <div
            className="h-full bg-primary rounded-full transition-all duration-1000 shadow-[0_0_20px_rgba(59,130,246,0.6)] relative overflow-hidden"
            style={{ width: `${task.progress || 0}%` }}
          >
            <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/30 to-transparent -translate-x-full animate-[shimmer_1.5s_infinite]" />
          </div>
        </div>
        <div className="absolute -top-1 opacity-0 group-hover/bar:opacity-100 transition-opacity right-0 -mr-2 w-1.5 h-4 bg-primary rounded-full blur-sm" />
      </div>
    </div>
  </div>
)

export default TaskRow
