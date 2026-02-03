import React, { useEffect } from 'react';
import {
    ChevronLeft,
    RefreshCcw,
    Folder,
    FileText,
    ChevronRight,
    ExternalLink,
    Trash2
} from 'lucide-react';
import { formatBytes } from '../utils/format';
import useExplorer from '../hooks/useExplorer';

const Explorer = ({ token }) => {
    const {
        explorerPath,
        explorerFiles,
        fetchExplorer,
        getExternalLink,
        deleteFile
    } = useExplorer(token);

    useEffect(() => {
        fetchExplorer(explorerPath);
    }, [explorerPath, fetchExplorer]);

    return (
        <div className="animate-in slide-in-from-right-10 duration-700 space-y-10">
            <div className="flex flex-col md:flex-row items-center justify-between glass-card p-10 rounded-[3rem] shadow-2xl gap-8">
                <div className="flex items-center space-x-8">
                    <button
                        onClick={() => {
                            const parts = explorerPath.split('/').filter(p => p !== '');
                            parts.pop();
                            fetchExplorer(parts.join('/'));
                        }}
                        disabled={!explorerPath}
                        className="p-5 bg-slate-100 dark:bg-white/5 rounded-2xl hover:bg-primary hover:text-white disabled:opacity-20 transition-all shadow-inner group"
                    >
                        <ChevronLeft size={28} className="group-hover:-translate-x-1 transition-transform" />
                    </button>
                    <div className="flex flex-col">
                        <h3 className="text-3xl font-[900] text-slate-900 dark:text-white">Distribution Explorer</h3>
                        <p className="text-[10px] font-black text-primary/60 mt-1.5 truncate max-w-md italic tracking-widest uppercase">root:/{explorerPath || ''}</p>
                    </div>
                </div>
                <button onClick={() => fetchExplorer(explorerPath)} className="p-4 bg-slate-50 dark:bg-white/5 text-slate-500 dark:text-slate-400 hover:text-primary rounded-2xl transition-all shadow-sm border border-slate-100 dark:border-white/5"><RefreshCcw size={24} /></button>
            </div>

            <div className="glass-card rounded-[3.5rem] overflow-hidden shadow-2xl border border-slate-100 dark:border-white/5 bg-white/80 dark:bg-zinc-900/40 backdrop-blur-3xl px-2">
                <div className="grid grid-cols-12 gap-4 px-10 py-8 bg-slate-50/20 dark:bg-white/5 border-b border-slate-100 dark:border-white/5 text-[11px] font-black uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">
                    <div className="col-span-6 md:col-span-7">Node Identifier</div>
                    <div className="col-span-3 md:col-span-2 text-center text-primary">Allocation</div>
                    <div className="col-span-3 text-right">Operations</div>
                </div>
                <div className="divide-y divide-slate-100 dark:divide-white/5">
                    {explorerFiles.map((file, i) => (
                        <div key={i} className="grid grid-cols-12 gap-5 px-10 py-7 items-center group hover:bg-slate-50/50 dark:hover:bg-white/[0.02] transition-all border-b border-slate-50 dark:border-white/5 last:border-0">
                            <div className="col-span-6 md:col-span-7 flex items-center space-x-6 min-w-0">
                                <div className={`p-4 rounded-[1.5rem] transition-all group-hover:scale-110 group-hover:rotate-6 shadow-sm ${file.isDir ? 'bg-primary/10 text-primary' : 'bg-slate-100 dark:bg-zinc-800 text-slate-500'}`}>
                                    {file.isDir ? <Folder size={24} /> : <FileText size={24} />}
                                </div>
                                <div className="flex-1 min-w-0">
                                    <p className="font-extrabold text-lg text-slate-900 dark:text-white truncate tracking-tighter transition-colors group-hover:text-primary" title={file.displayName || file.name}>
                                        {file.displayName || file.name}
                                    </p>
                                    <div className="flex items-center space-x-3 mt-1.5">
                                        <span className="text-[10px] font-black uppercase tracking-widest text-primary/70">Edge Link</span>
                                        <span className="w-1.5 h-1.5 rounded-full bg-slate-300 dark:bg-white/10" />
                                        <span className="text-[10px] font-bold text-slate-500 dark:text-slate-400 truncate opacity-70 font-mono italic">NODE-ID: {file.name.slice(0, 8).toUpperCase()}</span>
                                    </div>
                                </div>
                            </div>
                            <div className="col-span-3 md:col-span-2 text-center">
                                <span className="text-[11px] font-[900] text-slate-600 dark:text-slate-400 uppercase tracking-tighter bg-slate-110 dark:bg-white/5 px-3 py-1.5 rounded-lg">
                                    {file.isDir ? 'CONTAINER' : formatBytes(file.size)}
                                </span>
                            </div>
                            <div className="col-span-3 flex justify-end items-center space-x-3">
                                {file.isDir ? (
                                    <button onClick={() => fetchExplorer(explorerPath ? `${explorerPath}/${file.name}` : file.name)} className="p-3.5 bg-primary/10 text-primary rounded-2xl hover:bg-primary hover:text-white transition-all shadow-xl hover:shadow-primary/30 active:scale-90">
                                        <ChevronRight size={22} />
                                    </button>
                                ) : (
                                    <div className="flex items-center space-x-3">
                                        <button onClick={() => getExternalLink(file.name)} className="p-4 text-slate-500 dark:text-slate-400 hover:text-primary transition-all rounded-2xl hover:bg-slate-100 dark:hover:bg-white/10 hidden md:flex border border-transparent hover:border-primary/20">
                                            <ExternalLink size={22} />
                                        </button>
                                        <button onClick={() => deleteFile(file.name)} className="p-4 text-red-500 hover:text-white hover:bg-red-500 rounded-2xl transition-all shadow-xl hover:shadow-red-500/30 active:scale-95">
                                            <Trash2 size={22} />
                                        </button>
                                    </div>
                                )}
                            </div>
                        </div>
                    ))}
                </div>
            </div>
        </div>
    );
};

export default Explorer;
