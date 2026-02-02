import React from 'react';

const AnalyticsStat = ({ label, value, unit }) => (
    <div className="glass-card p-10 rounded-[3rem] text-center space-y-2 group hover:scale-[1.05] transition-all duration-500 shadow-xl">
        <p className="text-[10px] font-black text-slate-400 dark:text-zinc-500 uppercase tracking-widest">{label}</p>
        <p className="text-5xl font-black text-slate-900 dark:text-white tracking-tighter group-hover:text-primary transition-colors">{value}</p>
        <p className="text-[9px] font-black text-primary/60 uppercase tracking-[0.4em] pt-2">{unit}</p>
    </div>
);

export default AnalyticsStat;
