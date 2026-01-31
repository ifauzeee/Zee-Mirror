import React, { useState, useEffect } from 'react';
import {
    BarChart3,
    Activity,
    Settings,
    Folder,
    Download,
    Clock,
    HardDrive,
    Cpu,
    RefreshCcw,
    LogOut,
    ShieldCheck,
    Sun,
    Moon,
    Search,
    Trash2,
    FileText,
    ChevronLeft,
    ChevronRight,
    Save,
    AlertCircle,
    ExternalLink,
    Terminal,
    Github,
    Zap,
    Bot
} from 'lucide-react';
import axios from 'axios';

const Dashboard = () => {
    const [stats, setStats] = useState({ total_tasks: 0, total_bandwidth: 0, users_count: 0 });
    const [tasks, setTasks] = useState([]);
    const [system, setSystem] = useState({ cpu: 0, ram: 0, disk: 0, uptime: 0, os: '', arch: '' });
    const [settings, setSettings] = useState({ AutoDeleteMessages: false, DefaultMode: 'mirror' });
    const [loading, setLoading] = useState(true);
    const [activeTab, setActiveTab] = useState('overview');
    const [isDarkMode, setIsDarkMode] = useState(false);
    const [apiToken, setApiToken] = useState(localStorage.getItem('api_token') || 'zee-mirror-secret');
    const [loginError, setLoginError] = useState(false);

    const [explorerPath, setExplorerPath] = useState('');
    const [explorerFiles, setExplorerFiles] = useState([]);
    const [analyticsData, setAnalyticsData] = useState([]);
    const [logsContent, setLogsContent] = useState('');

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

    const fetchData = async () => {
        try {
            const config = { headers: { 'X-API-Key': apiToken } };
            const [statsRes, tasksRes, sysRes, settingsRes] = await Promise.all([
                axios.get('/api/stats', config),
                axios.get('/api/tasks', config),
                axios.get('/api/system', config),
                axios.get('/api/settings', config)
            ]);

            setStats(statsRes.data);
            setTasks(tasksRes.data || []);
            setSystem(sysRes.data);
            setSettings(settingsRes.data);
            setLoading(false);
            setLoginError(false);
        } catch (err) {
            if (err.response && err.response.status === 401) setLoginError(true);
        }
    };

    const fetchExplorer = async (path = '') => {
        try {
            const config = { headers: { 'X-API-Key': apiToken } };
            const res = await axios.get(`/api/explorer/remote?path=${encodeURIComponent(path)}`, config);

            const normalizedData = (res.data || []).map(item => ({
                name: item.Name,
                displayName: item.Name,
                isDir: item.IsDir,
                size: item.Size,
                time: item.ModTime,
                status: 'cloud'
            }));

            setExplorerFiles(normalizedData);
            setExplorerPath(path);
        } catch (err) {
            console.error("Explorer error:", err);
        }
    };

    const fetchAnalytics = async () => {
        try {
            const config = { headers: { 'X-API-Key': apiToken } };
            const res = await axios.get('/api/analytics', config);
            setAnalyticsData(res.data || []);
        } catch (err) {
            console.error("Analytics error:", err);
        }
    };

    const fetchLogs = async () => {
        try {
            const config = { headers: { 'X-API-Key': apiToken } };
            const res = await axios.get('/api/logs', config);
            setLogsContent(res.data.logs || '');
        } catch (err) {
            console.error("Logs error:", err);
        }
    };

    useEffect(() => {
        fetchData();
        const inv = setInterval(fetchData, 5000);
        return () => clearInterval(inv);
    }, [apiToken]);

    useEffect(() => {
        if (activeTab === 'files') fetchExplorer(explorerPath);
        if (activeTab === 'analytics') fetchAnalytics();
        if (activeTab === 'logs') fetchLogs();
    }, [activeTab, explorerPath]);

    const handleUpdateSettings = async (e) => {
        e.preventDefault();
        try {
            const config = { headers: { 'X-API-Key': apiToken } };
            await axios.post('/api/settings', settings, config);
            alert('Settings persistent storage updated!');
        } catch (err) {
            alert('Failed to update settings');
        }
    };

    const cancelTask = async (id) => {
        if (confirm('Cancel this task?')) {
            try {
                const config = { headers: { 'X-API-Key': apiToken } };
                await axios.delete(`/api/tasks?id=${id}`, config);
                fetchData();
            } catch (err) { console.error(err); }
        }
    };

    const getExternalLink = async (name) => {
        try {
            const config = { headers: { 'X-API-Key': apiToken } };
            const path = explorerPath ? `${explorerPath}/${name}` : name;
            const res = await axios.get(`/api/explorer/remote/link?path=${encodeURIComponent(path)}`, config);
            if (res.data.link) {
                window.open(res.data.link, '_blank');
            } else {
                alert('No public link available for this item');
            }
        } catch (err) {
            alert('Failed to generate cloud link');
        }
    };

    const deleteFile = async (name) => {
        if (confirm(`Permanently delete ${name} from Cloud Drive?`)) {
            try {
                const config = { headers: { 'X-API-Key': apiToken } };
                const path = explorerPath ? `${explorerPath}/${name}` : name;
                await axios.delete(`/api/explorer/remote?path=${encodeURIComponent(path)}`, config);
                fetchExplorer(explorerPath);
            } catch (err) { alert('Failed to delete cloud item'); }
        }
    };

    const formatBytes = (bytes) => {
        const num = parseFloat(bytes);
        if (isNaN(num) || num <= 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(num) / Math.log(k));
        return parseFloat((num / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    };

    if (loginError) {
        return (
            <div className="h-screen bg-slate-50 dark:bg-zinc-950 flex items-center justify-center p-6 text-slate-900 dark:text-white">
                <div className="bg-white dark:bg-zinc-900 p-10 rounded-[2.5rem] w-full max-w-md shadow-2xl border border-slate-200 dark:border-white/5">
                    <div className="flex flex-col items-center text-center space-y-6">
                        <div className="p-5 bg-red-500/10 rounded-full text-red-500 shadow-inner"><AlertCircle size={48} /></div>
                        <div>
                            <h2 className="text-3xl font-black mb-2">Access Portal</h2>
                            <p className="text-slate-500 text-sm font-medium">Please authenticate to access the Zee-Mirror Dashboard.</p>
                        </div>
                        <input
                            type="password"
                            placeholder="API Dashboard Token..."
                            className="w-full bg-slate-50 dark:bg-zinc-800 border border-slate-200 dark:border-zinc-700 p-4 rounded-2xl text-center font-bold outline-none focus:ring-2 ring-primary transition-all text-lg"
                            onKeyDown={(e) => {
                                if (e.key === 'Enter') {
                                    localStorage.setItem('api_token', e.target.value);
                                    setApiToken(e.target.value);
                                    window.location.reload();
                                }
                            }}
                        />
                        <p className="text-[10px] uppercase font-black tracking-widest text-slate-400">Default: zee-mirror-secret</p>
                    </div>
                </div>
            </div>
        );
    }

    return (
        <div className="flex h-screen bg-bg-light dark:bg-bg-dark text-text-main-light dark:text-text-main-dark overflow-hidden font-sans">
            <aside className="w-80 bg-white/80 dark:bg-zinc-950/50 backdrop-blur-2xl border-r border-slate-200 dark:border-white/5 flex flex-col p-8 space-y-10 z-30 shadow-2xl relative overflow-hidden">
                <div className="absolute inset-0 bg-gradient-to-b from-primary/5 to-transparent pointer-events-none" />

                <div className="flex items-center space-x-4 px-2 relative z-10">
                    <div className="p-3 premium-gradient rounded-2xl text-white shadow-lg shadow-primary/30 ring-4 ring-primary/10 group-hover:scale-110 transition-transform duration-500">
                        <Bot size={24} className="animate-pulse-soft" />
                    </div>
                    <div>
                        <h1 className="text-2xl font-black text-slate-900 dark:text-white tracking-[ -0.05em] uppercase leading-none">Zee<span className="text-primary italic">Mirror</span></h1>
                        <p className="text-[10px] font-black text-primary/60 tracking-[0.3em] uppercase mt-1">Cloud Engine</p>
                    </div>
                </div>

                <nav className="flex-1 space-y-3 relative z-10">
                    {[
                        { id: 'overview', label: 'Overview', icon: Activity },
                        { id: 'tasks', label: 'Tasks', icon: Download },
                        { id: 'files', label: 'Explorer', icon: Folder },
                        { id: 'analytics', label: 'Analytics', icon: BarChart3 },
                        { id: 'logs', label: 'Logs', icon: Terminal },
                        { id: 'settings', label: 'Settings', icon: Settings },
                    ].map(item => (
                        <button
                            key={item.id}
                            onClick={() => setActiveTab(item.id)}
                            className={`flex items-center space-x-4 w-full px-6 py-4 rounded-[1.5rem] transition-all duration-300 font-bold text-sm group ${activeTab === item.id
                                ? 'premium-gradient text-white shadow-2xl shadow-primary/30 scale-[1.02]'
                                : 'text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-white/5 hover:text-primary'
                                }`}
                        >
                            <item.icon size={20} className={activeTab === item.id ? '' : 'group-hover:scale-120 transition-transform'} />
                            <span className="tracking-tight">{item.label}</span>
                            {item.id === 'tasks' && tasks.length > 0 && (
                                <span className={`ml-auto px-2.5 py-1 rounded-full text-[10px] font-black ${activeTab === 'tasks' ? 'bg-white text-primary' : 'bg-primary/10 text-primary'}`}>
                                    {tasks.length}
                                </span>
                            )}
                        </button>
                    ))}
                </nav>

                <div className="mt-auto pt-8 border-t border-slate-200 dark:border-white/5 space-y-6 relative z-10">
                    <div className="p-5 bg-slate-50 dark:bg-zinc-900/50 rounded-[2rem] flex items-center space-x-4 border border-slate-100 dark:border-white/5 group transition-all hover:border-primary/30 shadow-inner">
                        <div className="w-12 h-12 rounded-2xl bg-primary/10 flex items-center justify-center text-primary group-hover:scale-110 transition-transform shadow-lg"><ShieldCheck size={24} /></div>
                        <div className="flex-1 min-w-0">
                            <p className="text-[11px] font-black uppercase tracking-widest text-slate-800 dark:text-white">Admin Access</p>
                            <div className="flex items-center text-[10px] text-green-500 font-black uppercase mt-0.5"><span className="w-1.5 h-1.5 rounded-full bg-green-500 mr-2 animate-pulse" />Online</div>
                        </div>
                        <button onClick={() => { localStorage.clear(); window.location.reload(); }} className="text-slate-400 hover:text-red-500 p-2 rounded-xl hover:bg-red-50 dark:hover:bg-red-500/10 transition-all"><LogOut size={20} /></button>
                    </div>

                    <div className="flex flex-col items-center space-y-3 pb-2">
                        <div className="text-center">
                            <p className="text-[8px] font-black uppercase tracking-[0.2em] text-slate-400">By : Ifauzeee</p>
                        </div>
                        <div className="flex space-x-3">
                            <a
                                href="https://github.com/ifauzeee/Zee-Mirror"
                                target="_blank"
                                rel="noopener noreferrer"
                                className="p-2 bg-slate-100 dark:bg-white/5 rounded-lg text-slate-500 hover:text-primary hover:bg-primary/10 transition-all border border-transparent hover:border-primary/20"
                                title="GitHub Repository"
                            >
                                <Github size={16} />
                            </a>
                            <a
                                href="https://github.com/ifauzeee"
                                target="_blank"
                                rel="noopener noreferrer"
                                className="p-2 bg-slate-100 dark:bg-white/5 rounded-lg text-slate-500 hover:text-primary hover:bg-primary/10 transition-all border border-transparent hover:border-primary/20"
                                title="Author Profile"
                            >
                                <ExternalLink size={16} />
                            </a>
                        </div>
                    </div>
                </div>
            </aside>

            <main className="flex-1 overflow-y-auto p-8 lg:p-12 scroll-smooth scrollbar-hide">
                <header className="flex flex-col md:flex-row justify-between items-start md:items-center mb-12 gap-6 animate-in fade-in slide-in-from-top-8 duration-700">
                    <div>
                        <div className="flex items-center space-x-3 mb-3">
                            <span className="w-12 h-1.5 bg-primary rounded-full shadow-[0_0_12px_rgba(59,130,246,0.5)]" />
                            <span className="text-[10px] font-black uppercase tracking-[0.4em] text-primary">System Dashboard v2.5</span>
                        </div>
                        <h2 className="text-5xl font-black text-slate-900 dark:text-white tracking-tighter mb-2 capitalize leading-tight">
                            {activeTab} <span className="text-primary italic">Panel</span>
                        </h2>
                        <p className="text-sm font-bold text-slate-400 max-w-lg leading-relaxed tracking-tight">
                            Managing high-performance cloud distribution & resilient local server resources in real-time.
                        </p>
                    </div>

                    <div className="flex items-center space-x-4">
                        <button onClick={toggleTheme} className="p-4 bg-white dark:bg-white/5 border border-slate-200 dark:border-white/5 rounded-[1.5rem] shadow-xl hover:scale-110 active:scale-95 transition-all text-slate-500 dark:text-yellow-400">
                            {isDarkMode ? <Sun size={22} /> : <Moon size={22} />}
                        </button>
                        <div className="px-8 py-4 premium-gradient text-white rounded-[1.5rem] font-black text-xs uppercase tracking-widest shadow-2xl shadow-primary/40 hover:scale-105 active:scale-95 transition-all group cursor-pointer flex items-center space-x-3">
                            <RefreshCcw size={18} className={loading ? 'animate-spin' : 'group-hover:rotate-[360deg] transition-transform duration-1000'} />
                            <span>{loading ? 'Syncing...' : 'Live Engine'}</span>
                        </div>
                    </div>
                </header>

                {activeTab === 'overview' && (
                    <div className="space-y-14 animate-in fade-in zoom-in-95 duration-700">
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
                            <StatsCard icon={Activity} label="Active Tasks" value={tasks.length} subLabel="Current Engine Queue" />
                            <StatsCard icon={Download} label="Total Mirror" value={stats.total_tasks} subLabel="Lifetime Success" />
                            <StatsCard icon={HardDrive} label="Network Traffic" value={formatBytes(stats.total_bandwidth)} subLabel="Total Throughput" />
                            <StatsCard icon={ShieldCheck} label="Access Control" value={stats.users_count || 1} subLabel="Authorized Nodes" />
                        </div>

                        <div className="grid grid-cols-1 xl:grid-cols-3 gap-12">
                            <div className="xl:col-span-2 space-y-8">
                                <div className="flex justify-between items-end">
                                    <h3 className="text-2xl font-black text-slate-900 dark:text-white flex items-center space-x-3">
                                        <div className="p-2 bg-primary/10 rounded-xl text-primary"><Clock size={20} /></div>
                                        <span>Process Monitor</span>
                                    </h3>
                                    <button onClick={() => setActiveTab('tasks')} className="px-5 py-2.5 bg-slate-100 dark:bg-white/5 rounded-xl text-[10px] font-black uppercase tracking-[0.2em] text-primary hover:bg-primary hover:text-white transition-all shadow-sm">
                                        View Full Buffer
                                    </button>
                                </div>
                                <div className="space-y-5">
                                    {(tasks.slice(0, 3)).length > 0 ? (
                                        <div className="grid grid-cols-1 gap-5">
                                            {tasks.slice(0, 3).map(task => (
                                                <TaskRow key={task.id} task={task} formatBytes={formatBytes} onCancel={cancelTask} />
                                            ))}
                                        </div>
                                    ) : (
                                        <div className="glass-card py-24 rounded-[3rem] flex flex-col items-center justify-center text-center group border-dashed border-2">
                                            <div className="p-10 bg-slate-50 dark:bg-white/5 rounded-full mb-6 group-hover:scale-110 transition-transform">
                                                <Activity size={64} className="text-slate-300 dark:text-slate-700 opacity-50" />
                                            </div>
                                            <h4 className="text-xl font-black text-slate-400 uppercase tracking-widest leading-loose">Engine Standing By<br /><span className="text-[10px] text-primary/40">Ready for distribution</span></h4>
                                        </div>
                                    )}
                                </div>
                            </div>
                            <div className="space-y-10">
                                <h3 className="text-2xl font-black text-slate-900 dark:text-white flex items-center space-x-3">
                                    <div className="p-2 bg-indigo-500/10 rounded-xl text-indigo-400"><Cpu size={20} /></div>
                                    <span>Infrastructure</span>
                                </h3>
                                <div className="glass-card p-12 rounded-[3.5rem] space-y-12 relative overflow-hidden">
                                    <div className="absolute top-0 right-0 p-6 opacity-5"><Zap size={100} /></div>
                                    <HealthBar label="CPU Load" value={Math.round(system.cpu)} />
                                    <HealthBar label="Memory Unit" value={Math.round(system.ram)} />
                                    <HealthBar label="Disk Matrix" value={Math.round(system.disk)} />

                                    <div className="pt-10 border-t border-slate-100 dark:border-white/5 space-y-6">
                                        <StatusItem label="Service Engine" value="Operational" color="text-green-500" />
                                        <StatusItem label="Environment" value={`${system.os || 'Linux'} ${system.arch || 'x64'}`} color="text-slate-500" />
                                        <StatusItem label="Active Uptime" value={system.uptime ? `${Math.floor(system.uptime / 3600)} Hours` : '99.9%'} color="text-primary" />
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                )}

                {activeTab === 'tasks' && (
                    <div className="animate-in slide-in-from-right-10 duration-700">
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-2 xl:grid-cols-3 gap-8">
                            {tasks.map(task => (
                                <TaskRow key={task.id} task={task} formatBytes={formatBytes} onCancel={cancelTask} />
                            ))}
                            {tasks.length === 0 && (
                                <div className="col-span-full py-20 text-center glass-card rounded-[3rem]">
                                    <Download size={48} className="mx-auto mb-4 text-slate-300" />
                                    <p className="font-bold text-slate-500">No active mirror or leech tasks.</p>
                                </div>
                            )}
                        </div>
                    </div>
                )}

                {activeTab === 'files' && (
                    <div className="animate-in slide-in-from-right-10 duration-700 space-y-8">
                        <div className="flex items-center justify-between glass-card p-8 rounded-[2.5rem] shadow-xl">
                            <div className="flex items-center space-x-6">
                                <button
                                    onClick={() => {
                                        const parts = explorerPath.split('/').filter(p => p !== '');
                                        parts.pop();
                                        fetchExplorer(parts.join('/'));
                                    }}
                                    disabled={!explorerPath}
                                    className="p-4 bg-slate-100 dark:bg-white/5 rounded-2xl hover:bg-primary hover:text-white disabled:opacity-20 transition-all shadow-inner"
                                >
                                    <ChevronLeft size={24} />
                                </button>
                                <div className="flex flex-col">
                                    <h3 className="text-2xl font-black text-slate-900 dark:text-white">Cloud Drive Explorer</h3>
                                    <p className="text-xs font-bold text-primary/60 mt-1 truncate max-w-md italic">gdrive:/{explorerPath || ''}</p>
                                </div>
                            </div>
                            <button onClick={() => fetchExplorer(explorerPath)} className="p-3.5 bg-slate-50 dark:bg-white/5 text-slate-400 hover:text-primary rounded-xl transition-all shadow-sm border border-slate-100 dark:border-white/5"><RefreshCcw size={20} /></button>
                        </div>

                        <div className="glass-card rounded-[2.5rem] overflow-hidden shadow-xl border border-slate-100 dark:border-white/5 bg-white dark:bg-zinc-900/50">
                            <div className="grid grid-cols-12 gap-4 px-8 py-5 bg-slate-50/50 dark:bg-white/5 border-b border-slate-100 dark:border-white/5 text-[10px] font-black uppercase tracking-widest text-slate-400">
                                <div className="col-span-6 md:col-span-7 flex items-center">Name</div>
                                <div className="col-span-3 md:col-span-2 text-center">Size</div>
                                <div className="col-span-3 text-right">Actions</div>
                            </div>
                            <div className="divide-y divide-slate-50 dark:divide-white/5">
                                {explorerFiles.map((file, i) => (
                                    <div key={i} className="grid grid-cols-12 gap-5 px-10 py-5 items-center group hover:bg-slate-50 dark:hover:bg-white/[0.02] transition-all border-b border-slate-50 dark:border-white/5 last:border-0">
                                        <div className="col-span-6 md:col-span-7 flex items-center space-x-5 min-w-0">
                                            <div className={`p-3 rounded-2xl transition-all group-hover:scale-110 shadow-sm ${file.isDir ? 'bg-primary/10 text-primary' : 'bg-slate-100 dark:bg-white/5 text-slate-400'}`}>
                                                {file.isDir ? <Folder size={20} /> : <FileText size={20} />}
                                            </div>
                                            <div className="flex-1 min-w-0">
                                                <p className="font-extrabold text-sm text-slate-900 dark:text-white truncate tracking-tight" title={file.displayName || file.name}>
                                                    {file.displayName || file.name}
                                                </p>
                                                <div className="flex items-center space-x-2.5 mt-1">
                                                    <span className="text-[10px] font-black uppercase tracking-widest text-primary/60">Cloud Node</span>
                                                    <span className="w-1 h-1 rounded-full bg-slate-300 dark:bg-white/10" />
                                                    <span className="text-[10px] font-bold text-slate-400 truncate opacity-60 font-mono italic">ID: {file.name.slice(0, 12)}...</span>
                                                </div>
                                            </div>
                                        </div>
                                        <div className="col-span-3 md:col-span-2 text-center">
                                            <span className="text-[10px] font-black text-slate-500 dark:text-slate-400 uppercase tracking-tighter">
                                                {file.isDir ? 'DIR' : formatBytes(file.size)}
                                            </span>
                                        </div>
                                        <div className="col-span-3 flex justify-end items-center space-x-2">
                                            {file.isDir ? (
                                                <button onClick={() => fetchExplorer(explorerPath ? `${explorerPath}/${file.name}` : file.name)} className="p-2.5 bg-primary/10 text-primary rounded-xl hover:bg-primary hover:text-white transition-all shadow-sm">
                                                    <ChevronRight size={18} />
                                                </button>
                                            ) : (
                                                <div className="flex items-center space-x-2">
                                                    <button onClick={() => getExternalLink(file.name)} className="p-2.5 text-slate-400 hover:text-primary transition-all rounded-xl hover:bg-slate-100 dark:hover:bg-white/10 hidden md:flex">
                                                        <ExternalLink size={18} />
                                                    </button>
                                                    <button onClick={() => deleteFile(file.name)} className="p-2.5 text-red-400 hover:text-white hover:bg-red-500 rounded-xl transition-all shadow-sm">
                                                        <Trash2 size={18} />
                                                    </button>
                                                </div>
                                            )}
                                        </div>
                                    </div>
                                ))}
                                {explorerFiles.length === 0 && (
                                    <div className="py-24 text-center">
                                        <Folder size={48} className="mx-auto mb-4 text-slate-200 dark:text-zinc-800" />
                                        <p className="text-slate-400 font-bold uppercase tracking-widest text-xs">Directory is empty</p>
                                    </div>
                                )}
                            </div>
                        </div>
                    </div>
                )}

                {activeTab === 'analytics' && (
                    <div className="animate-in slide-in-from-right-10 duration-700 space-y-12">
                        <h3 className="text-3xl font-black text-slate-900 dark:text-white">Productivity Trends</h3>
                        <div className="glass-card p-12 rounded-[3rem] h-[500px] flex items-end justify-between space-x-8 shadow-inner overflow-hidden relative">
                            <div className="absolute inset-0 grid grid-cols-1 grid-rows-4 pointer-events-none p-12">
                                {[1, 2, 3].map(v => <div key={v} className="border-t border-slate-100 dark:border-white/5 w-full h-full" />)}
                            </div>
                            {analyticsData.map((d, i) => (
                                <div key={i} className="flex-1 flex flex-col items-center group relative z-10">
                                    <div className="absolute -top-12 opacity-0 group-hover:opacity-100 transition-all bg-primary text-white text-[10px] font-bold px-2 py-1 rounded-lg">
                                        {d.TotalTasks} TASKS
                                    </div>
                                    <div
                                        className="w-full max-w-[40px] bg-gradient-to-t from-primary to-blue-400 rounded-t-2xl transition-all duration-1000 group-hover:scale-y-110 origin-bottom shadow-lg shadow-primary/20"
                                        style={{ height: `${Math.max((d.TotalTasks / (Math.max(...analyticsData.map(x => x.TotalTasks)) || 10)) * 300, 10)}px` }}
                                    />
                                    <p className="text-[10px] font-black text-text-sub-light dark:text-text-sub-dark uppercase mt-6 tracking-tighter">
                                        {new Date(d.Date).toLocaleDateString('en-US', { weekday: 'short' })}
                                    </p>
                                    <p className="text-xs font-black text-slate-900 dark:text-white mt-1">{new Date(d.Date).getDate()}</p>
                                </div>
                            ))}
                            {analyticsData.length === 0 && (
                                <div className="w-full h-full flex items-center justify-center">
                                    <BarChart3 size={64} className="text-slate-100 dark:text-zinc-800 animate-pulse" />
                                </div>
                            )}
                        </div>
                    </div>
                )}

                {activeTab === 'settings' && (
                    <div className="max-w-2xl animate-in slide-in-from-right-10 duration-700">
                        <form onSubmit={handleUpdateSettings} className="glass-card p-12 rounded-[3.5rem] space-y-10 group shadow-2xl">
                            <div className="flex items-center space-x-4 mb-4">
                                <div className="p-4 bg-primary rounded-[1.5rem] text-white shadow-xl shadow-primary/30"><Settings size={28} /></div>
                                <h3 className="text-3xl font-black text-slate-900 dark:text-white">Engine Settings</h3>
                            </div>

                            <div className="space-y-12">
                                <div className="flex items-center justify-between group/row">
                                    <div className="flex-1 pr-10">
                                        <h4 className="font-black text-lg text-slate-800 dark:text-slate-100 group-hover/row:text-primary transition-colors">Auto-Clear UI</h4>
                                        <p className="text-sm text-text-sub-light dark:text-text-sub-dark font-medium mt-1">Automatically wipe Telegram status messages after processing completes.</p>
                                    </div>
                                    <button
                                        type="button"
                                        onClick={() => setSettings({ ...settings, AutoDeleteMessages: !settings.AutoDeleteMessages })}
                                        className={`w-16 h-8 rounded-full transition-all duration-500 relative ring-4 ring-offset-4 dark:ring-offset-zinc-900 ${settings.AutoDeleteMessages ? 'bg-primary ring-primary/20' : 'bg-slate-200 dark:bg-zinc-800 ring-slate-100 dark:ring-zinc-800'}`}
                                    >
                                        <div className={`absolute top-1 w-6 h-6 bg-white rounded-full transition-all shadow-md ${settings.AutoDeleteMessages ? 'left-9 rotate-6' : 'left-1 -rotate-6'}`} />
                                    </button>
                                </div>

                                <div className="space-y-4">
                                    <h4 className="font-black text-lg text-slate-800 dark:text-slate-100">Default Upload Engine</h4>
                                    <div className="grid grid-cols-2 gap-4">
                                        {['mirror', 'leech'].map(mode => (
                                            <button
                                                key={mode}
                                                type="button"
                                                onClick={() => setSettings({ ...settings, DefaultMode: mode })}
                                                className={`p-6 rounded-3xl border-2 transition-all text-left group/btn ${settings.DefaultMode === mode ? 'border-primary bg-primary/5' : 'border-slate-100 dark:border-white/5 hover:border-slate-300'}`}
                                            >
                                                <div className={`w-10 h-10 rounded-xl mb-4 flex items-center justify-center transition-colors ${settings.DefaultMode === mode ? 'bg-primary text-white' : 'bg-slate-100 dark:bg-white/5 text-slate-400 group-hover/btn:text-primary'}`}>
                                                    {mode === 'mirror' ? <ExternalLink size={20} /> : <ShieldCheck size={20} />}
                                                </div>
                                                <p className="font-black uppercase text-xs tracking-widest">{mode}</p>
                                                <p className="text-[10px] text-text-sub-light mt-1 font-medium">{mode === 'mirror' ? 'Upload to Cloud' : 'Upload to Telegram'}</p>
                                            </button>
                                        ))}
                                    </div>
                                </div>
                            </div>

                            <button type="submit" className="w-full bg-primary text-white py-5 rounded-[2rem] font-black text-lg flex items-center justify-center space-x-3 shadow-2xl shadow-primary/40 hover:scale-[0.99] active:scale-[0.97] transition-all mt-10">
                                <Save size={24} />
                                <span>SYNCHRONIZE CORE</span>
                            </button>
                        </form>
                    </div>
                )}

                {activeTab === 'logs' && (
                    <div className="animate-in slide-in-from-right-10 duration-700 space-y-8">
                        <div className="flex items-center justify-between">
                            <h3 className="text-3xl font-black text-slate-900 dark:text-white">System Logs</h3>
                            <button onClick={fetchLogs} className="p-3.5 bg-slate-50 dark:bg-white/5 text-slate-400 hover:text-primary rounded-xl transition-all shadow-sm border border-slate-100 dark:border-white/5"><RefreshCcw size={20} /></button>
                        </div>
                        <div className="bg-zinc-950 rounded-[2.5rem] p-8 shadow-2xl border border-white/5 relative overflow-hidden group">
                            <div className="absolute top-0 left-0 w-full h-1.5 bg-gradient-to-r from-primary/50 via-blue-500/50 to-primary/50" />
                            <div className="flex items-center space-x-2 mb-6">
                                <div className="w-3 h-3 rounded-full bg-red-500/50" />
                                <div className="w-3 h-3 rounded-full bg-yellow-500/50" />
                                <div className="w-3 h-3 rounded-full bg-green-500/50" />
                                <span className="ml-4 text-[10px] font-black text-zinc-500 uppercase tracking-widest">zee-mirror.log — 100 lines</span>
                            </div>
                            <div className="font-mono text-[13px] text-zinc-300 overflow-y-auto max-h-[600px] scrollbar-hide leading-[1.1]">
                                {logsContent.split('\n').map((line, i) => (
                                    <div key={i} className="hover:bg-white/5 px-2 transition-colors group/line flex">
                                        <span className="text-zinc-700 mr-6 select-none group-hover/line:text-primary transition-colors inline-block w-8 text-right font-bold text-[10px] shrink-0 border-r border-white/5 h-full">{i + 1}</span>
                                        <span className={`whitespace-pre ${line.includes('ERROR') || line.includes('❌') ? 'text-red-400' : line.includes('⚠️') ? 'text-yellow-400' : 'text-zinc-300'}`}>
                                            {line || ' '}
                                        </span>
                                    </div>
                                ))}
                            </div>
                        </div>
                    </div>
                )}
            </main>
        </div>
    );
};

const StatsCard = ({ icon: Icon, label, value, subLabel }) => (
    <div className="glass-card p-10 rounded-[3rem] group hover:scale-[1.03] transition-all duration-500 hover:bg-slate-50 dark:hover:bg-white/[0.03] relative overflow-hidden">
        <div className="absolute top-0 right-0 p-4 opacity-5 group-hover:opacity-10 transition-opacity">
            <Icon size={120} className="-mr-10 -mt-10" />
        </div>
        <div className="w-16 h-16 rounded-[1.5rem] bg-primary/10 flex items-center justify-center mb-8 text-primary group-hover:bg-primary group-hover:text-white transition-all duration-500 shadow-inner relative z-10">
            <Icon size={32} />
        </div>
        <div className="space-y-2 relative z-10">
            <p className="text-[10px] font-black text-slate-400 uppercase tracking-[0.3em]">{label}</p>
            <p className="text-5xl font-black tracking-tighter text-slate-900 dark:text-white group-hover:text-primary transition-colors">{value}</p>
            <div className="flex items-center space-x-2">
                <span className="w-1 h-1 rounded-full bg-primary" />
                <p className="text-[9px] font-black text-primary/60 uppercase tracking-widest">{subLabel}</p>
            </div>
        </div>
    </div>
);

const TaskRow = ({ task, formatBytes, onCancel }) => (
    <div className="glass-card p-6 rounded-[2.5rem] flex flex-col space-y-5 hover:scale-[1.01] transition-all duration-300 border-l-4 border-l-primary group relative overflow-hidden">
        <div className="absolute top-0 right-0 p-3 opacity-10 group-hover:opacity-20 transition-opacity">
            <Download size={80} className="-mr-10 -mt-10 rotate-12" />
        </div>

        <div className="flex items-start justify-between relative z-10">
            <div className="flex items-center space-x-4 flex-1 min-w-0">
                <div className="p-3.5 bg-primary/10 text-primary rounded-2xl group-hover:bg-primary group-hover:text-white transition-all duration-500 shadow-inner shrink-0">
                    <Download size={22} className={task.status === 'downloading' ? 'animate-bounce' : ''} />
                </div>
                <div className="flex-1 min-w-0">
                    <div className="flex items-center space-x-2 mb-1">
                        <span className="px-2 py-0.5 bg-primary/10 text-primary text-[8px] font-black uppercase tracking-widest rounded-md shrink-0">
                            {task.type}
                        </span>
                        <span className="text-[10px] font-black text-primary/60 uppercase tracking-tighter truncate italic">
                            {task.status}
                        </span>
                    </div>
                    <h4 className="font-black text-base text-slate-900 dark:text-white truncate leading-tight pr-4" title={task.fileName || task.id}>
                        {task.fileName || `Task #${task.id}`}
                    </h4>
                </div>
            </div>
            <button
                onClick={() => onCancel(task.id)}
                className="p-3 text-slate-400 hover:text-white hover:bg-red-500 rounded-xl transition-all shrink-0 active:scale-90"
            >
                <Trash2 size={20} />
            </button>
        </div>

        <div className="space-y-4 relative z-10">
            <div className="flex items-end justify-between">
                <div className="space-y-1">
                    <p className="text-[10px] font-black text-slate-400 uppercase tracking-widest">Performance</p>
                    <div className="flex items-center space-x-3 text-xs font-bold text-slate-600 dark:text-slate-300">
                        <span className="flex items-center"><Activity size={12} className="mr-1 text-primary" /> {formatBytes(task.speed || 0)}/s</span>
                        <span className="text-slate-300 dark:text-white/10">|</span>
                        <span className="flex items-center"><Clock size={12} className="mr-1 text-indigo-400" /> {task.eta ? `${Math.floor(task.eta / 1e9 / 60)}m ${Math.floor(task.eta / 1e9) % 60}s` : '∞'}</span>
                    </div>
                </div>
                <div className="text-right">
                    <span className="text-2xl font-black text-primary font-mono tracking-tighter">
                        {(task.progress || 0).toFixed(1)}%
                    </span>
                </div>
            </div>

            <div className="relative">
                <div className="h-2 w-full bg-slate-100 dark:bg-white/5 rounded-full overflow-hidden shadow-inner flex p-0.5">
                    <div
                        className="h-full bg-primary rounded-full transition-all duration-1000 shadow-[0_0_15px_rgba(59,130,246,0.5)] relative overflow-hidden"
                        style={{ width: `${task.progress || 0}%` }}
                    >
                        <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/30 to-transparent -translate-x-full animate-[shimmer_2s_infinite]" />
                    </div>
                </div>
            </div>
        </div>
    </div>
);

const HealthBar = ({ label, value }) => (
    <div className="space-y-4">
        <div className="flex justify-between text-[11px] font-black text-slate-400 uppercase tracking-widest">
            <span>{label}</span>
            <span className="text-slate-900 dark:text-white font-mono">{value}%</span>
        </div>
        <div className="h-3 w-full bg-slate-100 dark:bg-zinc-800 rounded-full overflow-hidden p-0.5 shadow-inner">
            <div
                className={`h-full bg-primary rounded-full transition-all duration-1000 shadow-[0_0_15px_rgba(59,130,246,0.5)] relative overflow-hidden`}
                style={{ width: `${value}%` }}
            >
                <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent -translate-x-full animate-[shimmer_2s_infinite]" />
            </div>
        </div>
    </div>
);

const StatusItem = ({ label, value, color }) => (
    <div className="flex justify-between items-center group">
        <span className="text-[10px] font-black text-text-sub-light dark:text-text-sub-dark uppercase tracking-widest group-hover:text-primary transition-colors">{label}</span>
        <span className={`text-[11px] font-black tracking-tighter uppercase ${color}`}>{value}</span>
    </div>
);

export default Dashboard;
