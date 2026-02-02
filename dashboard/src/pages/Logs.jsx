import React, { useEffect } from 'react';
import { Terminal, RefreshCcw } from 'lucide-react';
import useLogs from '../hooks/useLogs';

const Logs = ({ token }) => {
    const { logsContent, fetchLogs } = useLogs(token);

    useEffect(() => {
        fetchLogs();
    }, [fetchLogs]);

    return (
        <div className="animate-in slide-in-from-right-10 duration-700 space-y-10">
            <div className="flex items-center justify-between px-2">
                <div className="flex items-center space-x-6">
                    <div className="p-4 bg-zinc-900 rounded-[1.5rem] text-zinc-400"><Terminal size={32} /></div>
                    <div>
                        <h3 className="text-4xl font-black text-slate-900 dark:text-white">Terminal Runtime</h3>
                        <p className="text-[10px] font-black uppercase tracking-[0.3em] text-primary mt-1">Direct Engine Output Interface</p>
                    </div>
                </div>
                <button onClick={fetchLogs} className="p-4 bg-slate-50 dark:bg-zinc-900 text-slate-400 hover:text-primary rounded-2xl transition-all shadow-sm border border-slate-100 dark:border-white/5"><RefreshCcw size={24} /></button>
            </div>
            <div className="bg-[#0c0c0c] rounded-[3rem] p-10 shadow-[0_40px_100px_-20px_rgba(0,0,0,0.4)] border border-white/5 overflow-hidden relative group">
                <div className="absolute top-0 left-0 w-full h-2 bg-gradient-to-r from-primary via-indigo-600 to-primary opacity-30 group-hover:opacity-100 transition-opacity duration-1000" />
                <div className="flex items-center justify-between mb-8 pb-6 border-b border-white/5">
                    <div className="flex space-x-2.5">
                        <div className="w-3.5 h-3.5 rounded-full bg-red-500/30 border border-red-500/20" />
                        <div className="w-3.5 h-3.5 rounded-full bg-yellow-500/30 border border-yellow-500/20" />
                        <div className="w-3.5 h-3.5 rounded-full bg-green-500/30 border border-green-500/20" />
                    </div>
                    <div className="flex items-center space-x-4">
                        <span className="px-3 py-1 bg-primary/20 text-primary text-[9px] font-black rounded-lg uppercase tracking-widest">Buffer Live</span>
                        <span className="text-[10px] font-black text-zinc-600 tracking-[0.2em]">RUNTIME.ZEE — {logsContent.split('\n').length} ENTRIES</span>
                    </div>
                </div>
                <div className="font-mono text-[14px] text-zinc-300 overflow-y-auto max-h-[650px] scrollbar-hide space-y-2 leading-relaxed selection:bg-primary selection:text-white">
                    {logsContent.split('\n').map((line, i) => (
                        <div key={i} className="hover:bg-white/[0.03] px-4 py-0.5 rounded-lg transition-colors group/line flex items-start gap-8 font-mono">
                            <span className="text-zinc-800 select-none group-hover/line:text-primary transition-colors text-right font-black text-[11px] shrink-0 mt-0.5 w-10">{(i + 1).toString().padStart(3, '0')}</span>
                            <span className={`break-words tracking-tight ${line.includes('ERROR') || line.includes('❌') ? 'text-red-400/90 font-bold' : line.includes('WARN') ? 'text-yellow-400/80' : 'text-zinc-400'}`}>
                                {line || ' '}
                            </span>
                        </div>
                    ))}
                </div>
            </div>
        </div>
    );
};

export default Logs;
