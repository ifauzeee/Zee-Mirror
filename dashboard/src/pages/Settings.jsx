import React, { useState, useEffect } from 'react';
import { Settings as SettingsIcon, Save, Folder, TrendingUp, ShieldCheck } from 'lucide-react';
import axios from 'axios';

const Settings = ({ token, initialSettings }) => {
    const [settings, setSettings] = useState(initialSettings || { AutoDeleteMessages: false, DefaultMode: 'mirror' });

    useEffect(() => {
        if (initialSettings) setSettings(initialSettings);
    }, [initialSettings]);

    const handleUpdateSettings = async (e) => {
        e.preventDefault();
        try {
            const config = { headers: { 'X-API-Key': token } };
            await axios.post('/api/settings', settings, config);
            alert('Settings updated!');
        } catch (err) {
            alert('Failed to update settings');
        }
    };

    return (
        <div className="max-w-3xl animate-in slide-in-from-right-10 duration-700">
            <form onSubmit={handleUpdateSettings} className="glass-card p-16 rounded-[4.5rem] space-y-12 shadow-[0_50px_100px_-20px_rgba(0,0,0,0.1)] border-[#ffffff50] dark:border-[#ffffff05] relative overflow-hidden group">
                <div className="absolute top-0 right-0 w-64 h-64 bg-primary/5 rounded-full blur-[80px] -mr-32 -mt-32" />

                <div className="flex items-center space-x-6 mb-4 relative z-10">
                    <div className="p-5 premium-gradient rounded-[2rem] text-white shadow-2xl shadow-primary/30 ring-8 ring-primary/5 group-hover:rotate-12 transition-transform duration-700"><SettingsIcon size={36} /></div>
                    <div>
                        <h3 className="text-4xl font-black text-slate-900 dark:text-white tracking-tighter">Cluster Protocol</h3>
                        <p className="text-[10px] font-black uppercase tracking-[0.4em] text-primary/60 mt-1">Global System Configuration</p>
                    </div>
                </div>

                <div className="space-y-14 relative z-10 pt-4">
                    <div className="flex items-center justify-between group/row p-8 rounded-[2.5rem] bg-slate-50/50 dark:bg-white/[0.02] border border-slate-100 dark:border-white/5 hover:border-primary/20 transition-all duration-500">
                        <div className="flex-1 pr-12">
                            <div className="flex items-center space-x-3 mb-2">
                                <div className="w-1.5 h-4 bg-primary rounded-full" />
                                <h4 className="font-black text-xl text-slate-900 dark:text-white">Ephemeral Buffer</h4>
                            </div>
                            <p className="text-sm text-slate-400 font-bold mt-2 leading-relaxed tracking-tight">Cascade delete Telegram status packets immediately after distribution cycle completes.</p>
                        </div>
                        <button
                            type="button"
                            onClick={() => setSettings({ ...settings, AutoDeleteMessages: !settings.AutoDeleteMessages })}
                            className={`w-20 h-10 rounded-full transition-all duration-700 relative p-1 shadow-inner ${settings.AutoDeleteMessages ? 'bg-primary' : 'bg-slate-200 dark:bg-zinc-800'}`}
                        >
                            <div className={`w-8 h-8 bg-white rounded-full transition-all duration-500 shadow-xl flex items-center justify-center ${settings.AutoDeleteMessages ? 'ml-10' : 'ml-0'}`}>
                                <div className={`w-1.5 h-1.5 rounded-full ${settings.AutoDeleteMessages ? 'bg-primary animate-pulse' : 'bg-slate-300'}`} />
                            </div>
                        </button>
                    </div>

                    <div className="space-y-8">
                        <div className="flex items-center space-x-3 px-2">
                            <div className="w-1.5 h-4 bg-indigo-500 rounded-full" />
                            <h4 className="font-black text-xl text-slate-900 dark:text-white uppercase tracking-wider">Default Transfer Mode</h4>
                        </div>
                        <div className="grid grid-cols-2 gap-8">
                            {['mirror', 'leech'].map(mode => (
                                <button
                                    key={mode}
                                    type="button"
                                    onClick={() => setSettings({ ...settings, DefaultMode: mode })}
                                    className={`p-10 rounded-[3rem] border-2 transition-all duration-500 text-left relative overflow-hidden group/btn ${settings.DefaultMode === mode ? 'border-primary bg-primary/5 shadow-2xl' : 'border-slate-100 dark:border-white/5 hover:border-primary/20 bg-white/5'}`}
                                >
                                    <div className={`w-14 h-14 rounded-2xl mb-8 flex items-center justify-center transition-all duration-700 ${settings.DefaultMode === mode ? 'bg-primary text-white scale-110 rotate-3 shadow-xl' : 'bg-slate-100 dark:bg-zinc-800 text-slate-400 group-hover/btn:scale-110'}`}>
                                        {mode === 'mirror' ? <Folder size={28} /> : <TrendingUp size={28} />}
                                    </div>
                                    <div className="relative z-10">
                                        <p className={`font-black uppercase text-xs tracking-[0.2em] mb-2 ${settings.DefaultMode === mode ? 'text-primary' : 'text-slate-400'}`}>{mode} protocol</p>
                                        <p className="text-xl font-black text-slate-900 dark:text-white tracking-tighter capitalize">{mode === 'mirror' ? 'Cloud Mirroring' : 'Resilient Leech'}</p>
                                        <p className="text-[10px] text-slate-400 mt-2 font-bold leading-relaxed">{mode === 'mirror' ? 'Execute high-speed upload to configured cloud storage providers.' : 'Deliver files directly back to Telegram encrypted data clusters.'}</p>
                                    </div>
                                    {settings.DefaultMode === mode && <div className="absolute top-6 right-6 p-2 bg-primary/10 rounded-full text-primary"><ShieldCheck size={20} /></div>}
                                </button>
                            ))}
                        </div>
                    </div>
                </div>

                <button type="submit" className="w-full bg-primary text-white py-6 rounded-[2.5rem] font-black text-xl flex items-center justify-center space-x-4 shadow-[0_24px_48px_-8px_rgba(59,130,246,0.6)] hover:scale-[0.98] active:scale-[0.95] transition-all duration-500 mt-14 relative overflow-hidden group">
                    <div className="absolute inset-0 bg-white/20 translate-y-full group-hover:translate-y-0 transition-transform duration-500" />
                    <Save size={28} className="group-hover:rotate-12 transition-transform" />
                    <span className="relative z-10 uppercase tracking-widest">Execute Core Synchronize</span>
                </button>
            </form>
        </div>
    );
};

export default Settings;
