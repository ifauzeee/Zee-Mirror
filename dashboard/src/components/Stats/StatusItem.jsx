const StatusItem = ({ label, value, color, dot }) => (
  <div className="flex justify-between items-center group/item hover:bg-slate-100 dark:hover:bg-white/[0.03] p-1.5 rounded-xl transition-all px-3">
    <div className="flex items-center space-x-3">
      {dot && <div className={`w-2 h-2 rounded-full ${dot} animate-pulse`} />}
      <span className="text-[11px] font-black text-slate-500 dark:text-zinc-500 uppercase tracking-widest group-hover/item:text-primary transition-colors">
        {label}
      </span>
    </div>
    <span
      className={`text-[11px] font-[900] tracking-tighter uppercase ${color || 'text-slate-900 dark:text-white'}`}
    >
      {value}
    </span>
  </div>
)

export default StatusItem
