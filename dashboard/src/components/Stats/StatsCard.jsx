const StatsCard = ({ icon: Icon, label, value, subLabel, color }) => (
  <div
    className={`glass-card p-10 rounded-[3.5rem] group hover:scale-[1.05] transition-all duration-700 hover:shadow-2xl relative overflow-hidden cursor-default border-t-2 ${color === 'primary' ? 'border-primary/40' : color === 'blue' ? 'border-blue-400/40' : color === 'indigo' ? 'border-indigo-400/40' : 'border-green-400/40'}`}
  >
    <div className="absolute top-[-40px] right-[-40px] w-48 h-48 premium-gradient rounded-full opacity-[0.03] group-hover:opacity-[0.08] transition-all duration-1000 blur-2xl group-hover:scale-150" />
    <div
      className={`w-16 h-16 rounded-2xl bg-${color}-500/10 flex items-center justify-center mb-8 text-${color}-500 group-hover:scale-110 group-hover:rotate-6 transition-all duration-700 shadow-inner ring-4 ring-${color}-500/5`}
    >
      {color === 'primary' ? <Icon size={32} className="text-primary" /> : <Icon size={32} />}
    </div>
    <div className="space-y-1 relative z-10">
      <p className="text-[9px] font-black text-slate-500 dark:text-zinc-500 uppercase tracking-[0.4em] mb-1 group-hover:text-primary transition-colors">
        {label}
      </p>
      <p className="text-4xl lg:text-5xl font-black tracking-[-0.05em] text-slate-900 dark:text-white leading-none mb-2">
        {value}
      </p>
      <div className="flex items-center space-x-2">
        <div
          className={`w-3 h-1 rounded-full bg-${color}-500 ${color === 'primary' ? 'bg-primary' : ''} group-hover:w-6 transition-all`}
        />
        <p className="text-[10px] font-black text-slate-500 dark:text-zinc-400 uppercase tracking-widest opacity-70 transition-opacity group-hover:opacity-100">
          {subLabel}
        </p>
      </div>
    </div>
  </div>
)

export default StatsCard
