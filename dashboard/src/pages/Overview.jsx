import {
  Activity,
  Download,
  HardDrive,
  ShieldCheck,
  TrendingUp,
  Clock,
  Layers,
  Zap,
} from 'lucide-react'
import StatsCard from '../components/Stats/StatsCard'
import TaskRow from '../components/Task/TaskRow'
import HealthBar from '../components/Stats/HealthBar'
import StatusItem from '../components/Stats/StatusItem'
import { formatBytes } from '../utils/format'

const Overview = ({ tasks, stats, system, onCancelTask, setActiveTab }) => {
  return (
    <div className="space-y-16 relative z-10 animate-in fade-in zoom-in-95 duration-1000">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
        <StatsCard
          icon={TrendingUp}
          label="Processing"
          value={tasks.length}
          subLabel="Active Node Latency"
          color="primary"
        />
        <StatsCard
          icon={Download}
          label="Deliveries"
          value={stats.total_tasks || 0}
          subLabel="Total Batch Success"
          color="blue"
        />
        <StatsCard
          icon={HardDrive}
          label="Throughput"
          value={formatBytes(stats.total_bandwidth || 0)}
          subLabel="Combined Data Flow"
          color="indigo"
        />
        <StatsCard
          icon={ShieldCheck}
          label="Authorized"
          value={stats.users_count || 1}
          subLabel="Secure Edge Points"
          color="green"
        />
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-12">
        <div className="xl:col-span-2 space-y-10">
          <div className="flex justify-between items-end px-2">
            <div className="flex items-center space-x-4">
              <div className="p-3 bg-primary/10 rounded-[1.25rem] text-primary">
                <Clock size={24} />
              </div>
              <h3 className="text-3xl font-black text-slate-900 dark:text-white tracking-tight">
                Active Processes
              </h3>
            </div>
            <button
              onClick={() => setActiveTab('tasks')}
              className="px-6 py-2.5 bg-slate-100 dark:bg-zinc-900/60 rounded-xl text-[9px] font-black uppercase tracking-widest text-primary hover:bg-primary hover:text-white transition-all shadow-sm border border-slate-200 dark:border-white/5"
            >
              View All Instances
            </button>
          </div>
          <div className="space-y-6">
            {tasks.length > 0 ? (
              <div className="grid grid-cols-1 gap-6">
                {tasks.slice(0, 3).map((task) => (
                  <TaskRow
                    key={task.id}
                    task={task}
                    formatBytes={formatBytes}
                    onCancel={onCancelTask}
                  />
                ))}
              </div>
            ) : (
              <div className="glass-card py-32 rounded-[3.5rem] flex flex-col items-center justify-center text-center group border-dashed border-2 border-slate-200 dark:border-white/10">
                <div className="p-12 bg-slate-100 dark:bg-white/5 rounded-full mb-8 group-hover:scale-110 transition-all duration-700 shadow-inner border border-slate-200 dark:border-transparent">
                  <Activity size={72} className="text-slate-400 dark:text-zinc-800 opacity-60" />
                </div>
                <h4 className="text-2xl font-black text-slate-600 dark:text-zinc-600 uppercase tracking-[0.2em] leading-relaxed">
                  Cluster Idle
                  <br />
                  <span className="text-[10px] text-primary/60 tracking-[0.5em]">
                    Waiting for distribution protocols
                  </span>
                </h4>
              </div>
            )}
          </div>
        </div>

        <div className="space-y-12">
          <div className="flex items-center space-x-4 px-2">
            <div className="p-3 bg-indigo-500/10 rounded-[1.25rem] text-indigo-400">
              <Layers size={24} />
            </div>
            <h3 className="text-3xl font-black text-slate-900 dark:text-white tracking-tight">
              Node Metrics
            </h3>
          </div>
          <div className="glass-card p-12 rounded-[4rem] space-y-12 relative overflow-hidden group">
            <div className="absolute top-0 right-0 p-8 opacity-[0.03] group-hover:opacity-[0.07] transition-all duration-1000 rotate-12">
              <Zap size={140} />
            </div>
            <div className="space-y-12 relative z-10">
              <HealthBar
                label="Global CPU Load"
                value={Math.round(system.cpu || 0)}
                color="from-primary to-blue-400"
              />
              <HealthBar
                label="Memory Utilization"
                value={Math.round(system.ram || 0)}
                color="from-indigo-500 to-purple-400"
              />
              <HealthBar
                label="Storage Allocation"
                value={Math.round(system.disk || 0)}
                color="from-blue-600 to-cyan-400"
              />
            </div>

            <div className="pt-12 border-t border-slate-100 dark:border-white/5 space-y-7 relative z-10">
              <StatusItem label="Edge Service" value="Operational" dot="bg-green-500" />
              <StatusItem
                label="Architecture"
                value={`${system.os || 'Linux'} ${system.arch || 'x64'}`}
                color="text-slate-600"
              />
              <StatusItem
                label="Active Uptime"
                value={
                  system.uptime
                    ? `${Math.floor(system.uptime / 3600)}h ${Math.floor((system.uptime % 3600) / 60)}m`
                    : 'Stable'
                }
                color="text-primary"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default Overview
