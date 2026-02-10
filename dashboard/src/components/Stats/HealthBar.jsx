const HealthBar = ({ label, value, color }) => (
  <div className="space-y-4 group">
    <div className="flex justify-between text-[11px] font-[900] text-slate-500 dark:text-zinc-500 uppercase tracking-[0.3em] group-hover:text-primary transition-colors px-1">
      <span>{label}</span>
      <span className="text-slate-900 dark:text-white font-black tracking-widest">
        {value}
        <span className="text-[8px] ml-0.5 opacity-40">%</span>
      </span>
    </div>
    <div className="h-3.5 w-full bg-slate-100 dark:bg-zinc-800 rounded-full overflow-hidden p-0.5 shadow-inner border border-slate-100 dark:border-white/5">
      <div
        className={`h-full bg-gradient-to-r ${color} rounded-full transition-all duration-1000 shadow-[0_0_20px_rgba(59,130,246,0.5)] relative overflow-hidden lg:animate-pulse-slow`}
        style={{ width: `${value}%` }}
      >
        <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent -translate-x-full animate-[shimmer_3s_infinite]" />
      </div>
    </div>
  </div>
)

export default HealthBar
