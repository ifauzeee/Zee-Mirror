import React, { useState, useEffect } from 'react';
import { RefreshCcw, Sun, Moon, ArrowRight, Bot, Activity, ShieldCheck, Lock } from 'lucide-react';

import Sidebar from './components/Sidebar/Sidebar';
import Overview from './pages/Overview';
import Explorer from './pages/Explorer';
import { usePopups } from './context/PopupContext';
import Settings from './pages/Settings';
import Analytics from './pages/Analytics';
import Logs from './pages/Logs';
import TorrentSelect from './pages/TorrentSelect';
import Users from './pages/Users';
import TaskRow from './components/Task/TaskRow';

import useTasks from './hooks/useTasks';
import useSystemStats from './hooks/useSystemStats';
import axios from 'axios';


const LoginScreen = ({ setApiToken, setLoginError }) => {
    const [authStep, setAuthStep] = useState('welcome');

    return (
        <div className="min-h-screen w-full bg-white dark:bg-[#050505] flex flex-col md:flex-row font-sans overflow-hidden">
            <div className="md:w-1/2 lg:w-3/5 h-[60vh] md:h-screen premium-gradient p-12 md:p-24 text-white flex flex-col relative overflow-hidden shrink-0">
                <div className="absolute inset-0 opacity-20 pointer-events-none">
                    <div className="absolute inset-0 bg-[url('https://www.transparenttextures.com/patterns/carbon-fibre.png')] opacity-10" />
                    <div className="absolute top-[-20%] right-[-20%] w-[100%] h-[100%] bg-white/10 rounded-full blur-[180px]" />
                    <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 opacity-[0.02] scale-[6] rotate-12 transition-transform duration-[10s] animate-pulse">
                        <Bot size={400} />
                    </div>
                </div>

                <div className="relative z-10 flex items-center space-x-6">
                    <div className="p-4 bg-white/10 backdrop-blur-3xl border border-white/20 rounded-3xl w-fit shadow-2xl ring-8 ring-white/5 transition-all hover:scale-105">
                        <Bot size={40} className="text-white" />
                    </div>
                    <div className="h-[1px] w-12 bg-white/20 rounded-full" />
                    <span className="text-[10px] font-black uppercase tracking-[0.8em] opacity-40">System Core</span>
                </div>

                <div className="relative z-10 my-auto py-16">
                    <div className="space-y-6">
                        <h2 className="text-[90px] lg:text-[140px] font-[1000] tracking-[-0.06em] leading-[0.75] uppercase flex flex-col">
                            <span className="animate-in fade-in slide-in-from-left-12 duration-1000">Zee</span>
                            <span className="text-white/25 italic animate-in fade-in slide-in-from-left-16 duration-1000 delay-200">Mirror</span>
                        </h2>
                        <div className="h-[5px] w-24 bg-primary/50 rounded-full shadow-[0_0_20px_rgba(59,130,246,0.5)]" />
                    </div>
                    <p className="text-white/80 text-xl lg:text-3xl font-bold leading-relaxed max-w-lg mt-12 opacity-80 animate-in fade-in slide-in-from-bottom-8 duration-1000 delay-500">
                        Orchestrating the next generation of resilient cloud orchestration & global distribution networks.
                    </p>
                </div>

                <div className="relative z-10 grid grid-cols-1 sm:grid-cols-2 gap-10 pt-10 border-t border-white/5 mt-auto">
                    <div className="flex items-center space-x-5 group cursor-default">
                        <div className="w-16 h-16 rounded-[2rem] bg-white/5 flex items-center justify-center backdrop-blur-2xl border border-white/10 shadow-2xl shrink-0 group-hover:bg-white/10 transition-all duration-500">
                            <Activity size={28} className="text-green-400" />
                        </div>
                        <div>
                            <p className="text-[10px] font-black uppercase tracking-[0.4em] text-white/30 mb-2 leading-none">Global Engine</p>
                            <p className="text-sm font-black uppercase tracking-widest text-white leading-none">Sync Status: <span className="text-green-400">Online</span></p>
                        </div>
                    </div>
                    <div className="flex items-center space-x-5 group cursor-default">
                        <div className="w-16 h-16 rounded-[2rem] bg-white/5 flex items-center justify-center backdrop-blur-2xl border border-white/10 shadow-2xl shrink-0 group-hover:bg-white/10 transition-all duration-500">
                            <ShieldCheck size={28} className="text-primary" />
                        </div>
                        <div>
                            <p className="text-[10px] font-black uppercase tracking-[0.4em] text-white/30 mb-2 leading-none">Security Protocol</p>
                            <p className="text-sm font-black uppercase tracking-widest text-white leading-none">Authorization: <span className="text-primary italic font-black">Secure</span></p>
                        </div>
                    </div>
                </div>
            </div>

            <div className="flex-1 min-h-screen flex flex-col items-center justify-center p-12 md:p-24 relative bg-[#fcfdfe] dark:bg-zinc-950/40">
                <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,_var(--tw-gradient-stops))] from-primary/10 via-transparent to-transparent opacity-40" />

                <div className="w-full max-w-sm relative z-10 -mt-12">
                    {authStep === 'welcome' ? (
                        <div className="space-y-16 animate-in fade-in slide-in-from-bottom-8 duration-700">
                            <div className="space-y-8 text-center md:text-left">
                                <div className="flex items-center justify-center md:justify-start space-x-4">
                                    <div className="h-[2px] w-12 bg-primary rounded-full shadow-[0_0_15px_rgba(59,130,246,0.3)]" />
                                    <span className="text-[11px] font-black text-primary uppercase tracking-[0.8em]">Entry Protocol</span>
                                </div>
                                <h3 className="text-8xl font-[1000] text-slate-900 dark:text-white tracking-[-0.08em] leading-none">Hello.</h3>
                                <p className="text-slate-400 dark:text-zinc-500 font-bold text-xl leading-relaxed max-w-[280px] mx-auto md:mx-0">System standby. Provide identity to bridge link.</p>
                            </div>
                            <button
                                onClick={() => setAuthStep('password')}
                                className="w-full p-10 premium-gradient rounded-[3rem] text-white font-black uppercase tracking-[0.5em] text-lg shadow-[0_32px_64px_-12px_rgba(59,130,246,0.5)] hover:shadow-primary/70 hover:scale-[1.05] active:scale-95 transition-all duration-500 flex items-center justify-center group"
                            >
                                <span>Enter Engine</span>
                                <ArrowRight size={28} className="ml-6 group-hover:translate-x-3 transition-transform" />
                            </button>
                        </div>
                    ) : (
                        <div className="space-y-12 animate-in fade-in slide-in-from-bottom-8 duration-700">
                            <div className="mb-16 flex justify-center md:justify-start">
                                <button
                                    onClick={() => setAuthStep('welcome')}
                                    className="p-5 bg-slate-100 dark:bg-white/5 rounded-3xl text-slate-400 dark:text-zinc-600 hover:text-primary transition-all flex items-center group shadow-sm"
                                >
                                    <ArrowRight size={20} className="rotate-180 group-hover:-translate-x-2 transition-transform" />
                                </button>
                            </div>
                            <div className="space-y-8 text-center md:text-left">
                                <div className="flex items-center justify-center md:justify-start space-x-4">
                                    <div className="h-[2px] w-12 bg-red-500 rounded-full shadow-[0_0_10px_rgba(239,68,68,0.4)]" />
                                    <span className="text-[11px] font-black text-red-500 uppercase tracking-[0.8em]">Security Level 5</span>
                                </div>
                                <h3 className="text-7xl font-[1000] tracking-[-0.06em] leading-none text-slate-900 dark:text-white">Secret Key</h3>
                                <p className="text-slate-400 dark:text-zinc-500 font-bold text-xl leading-relaxed max-w-[320px] mx-auto md:mx-0">System demands the encrypted synchronization token.</p>
                            </div>
                            <div className="space-y-8">
                                <div className="relative group">
                                    <input
                                        type="password"
                                        placeholder="SYNC_SECRET_KEY"
                                        autoFocus
                                        className="w-full bg-slate-50 dark:bg-white/5 border border-slate-200 dark:border-white/5 p-12 rounded-[3.5rem] text-center font-black outline-none focus:ring-[16px] ring-primary/10 transition-all text-2xl tracking-[0.2em] text-slate-950 dark:text-white placeholder:text-slate-300 dark:placeholder:text-zinc-900 group-hover:bg-white dark:group-hover:bg-white/10"
                                        onKeyDown={(e) => {
                                            if (e.key === 'Enter') {
                                                const val = e.target.value;
                                                localStorage.setItem('api_token', val);
                                                setApiToken(val);
                                                setLoginError(false);
                                                window.location.reload();
                                            }
                                        }}
                                    />
                                </div>
                                <div className="flex items-center justify-center space-x-4 text-slate-300 dark:text-zinc-700">
                                    <Lock size={16} className="animate-pulse" />
                                    <p className="text-[10px] font-black uppercase tracking-[0.6em] italic">Established Encrypted Link</p>
                                </div>
                            </div>
                        </div>
                    )}
                </div>

                <div className="absolute bottom-12 flex items-center space-x-6 opacity-30">
                    <p className="text-[10px] font-black text-slate-400 dark:text-zinc-600 uppercase tracking-[0.5em]">
                        Built for Resilient Distributions
                    </p>
                </div>
            </div>
        </div>
    );
};

const Dashboard = () => {
    const [apiToken, setApiToken] = useState(localStorage.getItem('api_token') || '');
    const [loginError, setLoginError] = useState(!localStorage.getItem('api_token'));
    const [activeTab, setActiveTab] = useState('overview');
    const [isDarkMode, setIsDarkMode] = useState(false);
    const [loading, setLoading] = useState(false);


    const { tasks, fetchTasks, cancelTask, setTasks } = useTasks(apiToken);
    const { stats, system, fetchStats, setSystem } = useSystemStats(apiToken);
    const { showConfirm } = usePopups();
    const [settings, setSettings] = useState(null);


    useEffect(() => {
        const interceptor = axios.interceptors.response.use(
            (response) => response,
            (error) => {
                if (error.response && error.response.status === 401) {
                    setLoginError(true);
                    setApiToken('');
                    localStorage.removeItem('api_token');
                }
                return Promise.reject(error);
            }
        );

        return () => {
            axios.interceptors.response.eject(interceptor);
        };
    }, []);

    useEffect(() => {
        const savedTheme = localStorage.getItem('theme') || 'light';
        const isDark = savedTheme === 'dark';
        setIsDarkMode(isDark);
        if (isDark) document.documentElement.classList.add('dark');
        else document.documentElement.classList.remove('dark');
    }, []);

    const toggleTheme = () => {
        const newMode = !isDarkMode;
        setIsDarkMode(newMode);
        localStorage.setItem('theme', newMode ? 'dark' : 'light');
        if (newMode) document.documentElement.classList.add('dark');
        else document.documentElement.classList.remove('dark');
    };


    useEffect(() => {
        if (apiToken && !loginError) {
            setLoading(true);
            const loadData = async () => {
                try {
                    await Promise.all([fetchTasks(), fetchStats()]);

                    try {
                        const res = await axios.get('/api/settings', { headers: { 'X-API-Key': apiToken } });
                        setSettings(res.data);
                    } catch (e) {
                        console.error("Settings load error", e);
                    }
                } catch (e) {
                    console.error("Initial load failed", e);
                } finally {
                    setLoading(false);
                }
            };
            loadData();

            let ws;
            const connectWs = () => {
                const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
                const wsUrl = `${protocol}//${window.location.host}/api/ws?token=${apiToken}`;
                ws = new WebSocket(wsUrl);

                ws.onopen = () => console.log('🟢 WS Connected');

                ws.onmessage = (event) => {
                    try {
                        const msg = JSON.parse(event.data);
                        if (msg.type === 'update') {
                            if (msg.data.tasks) setTasks(msg.data.tasks);
                            if (msg.data.system) setSystem(prev => ({ ...prev, ...msg.data.system }));
                        }
                    } catch (e) { console.error('WS Parse Error', e); }
                };

                ws.onclose = () => {
                    console.log('🔴 WS Closed, reconnecting in 3s...');
                    setTimeout(() => {
                        if (apiToken && !loginError) connectWs();
                    }, 3000);
                };
            };
            connectWs();

            return () => {
                if (ws) ws.close();
            };
        }
    }, [apiToken, loginError, fetchTasks, fetchStats, setTasks, setSystem]);

    const isTorrentSelectPage = window.location.pathname.startsWith('/torrent-select/');

    if (isTorrentSelectPage) {
        return <TorrentSelect token={apiToken} />;
    }

    if (loginError) {
        return <LoginScreen setApiToken={setApiToken} setLoginError={setLoginError} />;
    }

    const formatBytes = (bytes) => {

        const num = parseFloat(bytes);
        if (isNaN(num) || num <= 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(num) / Math.log(k));
        return parseFloat((num / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    };

    return (
        <div className="flex h-screen bg-[#f8fafc] dark:bg-[#050505] text-[#0f172a] dark:text-[#f8fafc] overflow-hidden font-sans selection:bg-primary selection:text-white">
            <Sidebar
                activeTab={activeTab}
                setActiveTab={setActiveTab}
                tasksCount={tasks.length}
                onLogout={async () => {
                    if (await showConfirm('Secure Disconnect', 'Are you sure you want to disconnect from the Cloud Engine? Active sessions will be terminated.')) {
                        localStorage.clear();
                        window.location.reload();
                    }
                }}
            />

            <main className="flex-1 overflow-y-auto p-12 scroll-smooth scrollbar-hide relative">
                <div className="absolute top-[-100px] right-[-100px] w-[500px] h-[500px] bg-primary/5 rounded-full blur-[120px] pointer-events-none" />
                <div className="absolute bottom-[-100px] left-[-100px] w-[500px] h-[500px] bg-indigo-500/5 rounded-full blur-[120px] pointer-events-none" />

                <header className="flex flex-col md:flex-row justify-between items-start md:items-center mb-16 gap-8 relative z-20">
                    <div className="animate-in fade-in slide-in-from-left-8 duration-700">
                        <div className="flex items-center space-x-4 mb-4">
                            <div className="h-1.5 w-12 bg-primary rounded-full shadow-[0_0_12px_rgba(59,130,246,0.5)]" />
                            <span className="text-[10px] font-black uppercase tracking-[0.4em] text-primary opacity-80">Cloud Infrastructure Matrix</span>
                        </div>
                        <h2 className="text-6xl font-black text-slate-900 dark:text-white tracking-[-0.04em] mb-3 capitalize leading-[0.9]">
                            {activeTab} <span className="text-primary italic">Engine</span>
                        </h2>
                        <p className="text-sm font-bold text-slate-400 max-w-lg leading-relaxed tracking-tight group">
                            Monitoring high-performance distribution clusters & edge computing resources in <span className="text-slate-600 dark:text-slate-200 transition-colors">real-time globally</span>.
                        </p>
                    </div>

                    <div className="flex items-center space-x-5 animate-in fade-in slide-in-from-right-8 duration-700">
                        <button onClick={toggleTheme} className="p-4 bg-white dark:bg-zinc-900/60 border border-slate-200 dark:border-white/5 rounded-[1.75rem] shadow-xl hover:scale-110 active:scale-95 transition-all text-slate-500 dark:text-yellow-400 hover:shadow-primary/10">
                            {isDarkMode ? <Sun size={24} /> : <Moon size={24} />}
                        </button>
                        <div className="px-8 py-4 premium-gradient text-white rounded-[1.75rem] font-black text-[10px] uppercase tracking-[0.2em] shadow-[0_12px_32px_-8px_rgba(59,130,246,0.5)] hover:scale-105 active:scale-95 transition-all group flex items-center space-x-4">
                            <RefreshCcw size={18} className={loading ? 'animate-spin' : 'group-hover:rotate-180 transition-transform duration-700'} />
                            <span>{loading ? 'Refreshing...' : 'Cluster Active'}</span>
                        </div>
                    </div>
                </header>

                {activeTab === 'overview' && (
                    <Overview
                        tasks={tasks}
                        stats={stats}
                        system={system}
                        onCancelTask={cancelTask}
                        setActiveTab={setActiveTab}
                    />
                )}

                {activeTab === 'tasks' && (
                    <div className="space-y-6 animate-in slide-in-from-right-10 duration-700">
                        {tasks.length > 0 ? (
                            <div className="grid grid-cols-1 gap-6">
                                {tasks.map(task => (
                                    <TaskRow key={task.id} task={task} formatBytes={formatBytes} onCancel={cancelTask} />
                                ))}
                            </div>
                        ) : (
                            <div className="text-center text-slate-400 mt-20">No active tasks</div>
                        )}
                    </div>
                )}

                {activeTab === 'files' && <Explorer token={apiToken} />}
                {activeTab === 'analytics' && <Analytics token={apiToken} isDarkMode={isDarkMode} />}
                {activeTab === 'logs' && <Logs token={apiToken} />}
                {activeTab === 'users' && <Users apiToken={apiToken} />}
                {activeTab === 'settings' && <Settings token={apiToken} initialSettings={settings} />}

            </main>
        </div>
    );
};

export default Dashboard;
