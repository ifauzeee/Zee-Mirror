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
    Bot,
    TrendingUp,
    Layers,
    ArrowRight,
    Lock
} from 'lucide-react';
import axios from 'axios';
import {
    AreaChart,
    Area,
    XAxis,
    YAxis,
    CartesianGrid,
    Tooltip,
    ResponsiveContainer,
    BarChart,
    Bar,
    Cell
} from 'recharts';

const Dashboard = () => {
    const [stats, setStats] = useState({ total_tasks: 0, total_bandwidth: 0, users_count: 0 });
    const [tasks, setTasks] = useState([]);
    const [system, setSystem] = useState({ cpu: 0, ram: 0, disk: 0, uptime: 0, os: '', arch: '' });
    const [settings, setSettings] = useState({ AutoDeleteMessages: false, DefaultMode: 'mirror' });
    const [loading, setLoading] = useState(true);
    const [activeTab, setActiveTab] = useState('overview');
    const [isDarkMode, setIsDarkMode] = useState(false);
    const [apiToken, setApiToken] = useState(localStorage.getItem('api_token') || '');
    const [loginError, setLoginError] = useState(!localStorage.getItem('api_token'));

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

    const handleRequest = (promise) => promise.catch(err => {
        if (err.response && err.response.status === 401) throw err;
        return { data: {} };
    });

    const fetchData = async () => {
        try {
            const config = { headers: { 'X-API-Key': apiToken } };
            const [statsRes, tasksRes, sysRes, settingsRes] = await Promise.all([
                handleRequest(axios.get('/api/stats', config)),
                handleRequest(axios.get('/api/tasks', config)),
                handleRequest(axios.get('/api/system', config)),
                handleRequest(axios.get('/api/settings', config))
            ]);

            if (statsRes.data) setStats(statsRes.data);
            setTasks(tasksRes.data || []);
            if (sysRes.data) setSystem(sysRes.data);
            if (settingsRes.data) setSettings(settingsRes.data);
            setLoading(false);
            setLoginError(false);
        } catch (err) {
            if (err.response && err.response.status === 401) {
                setLoginError(true);
                setAuthStep('welcome'); // Reset auth step on error
            }
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
            if (err.response && err.response.status === 401) setLoginError(true);
        }
    };

    const fetchAnalytics = async () => {
        try {
            const config = { headers: { 'X-API-Key': apiToken } };
            const res = await axios.get('/api/analytics', config);
            setAnalyticsData(res.data || []);
        } catch (err) {
            console.error("Analytics error:", err);
            if (err.response && err.response.status === 401) setLoginError(true);
        }
    };

    const fetchLogs = async () => {
        try {
            const config = { headers: { 'X-API-Key': apiToken } };
            const res = await axios.get('/api/logs', config);
            setLogsContent(res.data.logs || '');
        } catch (err) {
            console.error("Logs error:", err);
            if (err.response && err.response.status === 401) setLoginError(true);
        }
    };

    useEffect(() => {
        if (apiToken && !loginError) {
            fetchData();
            const inv = setInterval(fetchData, 5000);
            return () => clearInterval(inv);
        } else {
            setLoading(false);
        }
    }, [apiToken, loginError]);

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
            alert('Settings updated!');
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
                alert('No public link available');
            }
        } catch (err) { alert('Failed to generate cloud link'); }
    };

    const deleteFile = async (name) => {
        if (confirm(`Permanently delete ${name}?`)) {
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

    const [authStep, setAuthStep] = useState('welcome');

    if (loginError) {
        return (
            <div className="min-h-screen w-full bg-white dark:bg-[#050505] flex flex-col md:flex-row font-sans overflow-hidden">
                {/* Left Panel: High-End Cinematic Identity */}
                <div className="md:w-1/2 lg:w-3/5 h-[60vh] md:h-screen premium-gradient p-12 md:p-24 text-white flex flex-col relative overflow-hidden shrink-0">
                    {/* Artistic Background Layer */}
                    <div className="absolute inset-0 opacity-20 pointer-events-none">
                        <div className="absolute inset-0 bg-[url('https://www.transparenttextures.com/patterns/carbon-fibre.png')] opacity-10" />
                        <div className="absolute top-[-20%] right-[-20%] w-[100%] h-[100%] bg-white/10 rounded-full blur-[180px]" />
                        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 opacity-[0.02] scale-[6] rotate-12 transition-transform duration-[10s] animate-pulse">
                            <Bot size={400} />
                        </div>
                    </div>

                    {/* Header: Identity Stamp */}
                    <div className="relative z-10 flex items-center space-x-6">
                        <div className="p-4 bg-white/10 backdrop-blur-3xl border border-white/20 rounded-3xl w-fit shadow-2xl ring-8 ring-white/5 transition-all hover:scale-105">
                            <Bot size={40} className="text-white" />
                        </div>
                        <div className="h-[1px] w-12 bg-white/20 rounded-full" />
                        <span className="text-[10px] font-black uppercase tracking-[0.8em] opacity-40">System Core</span>
                    </div>

                    {/* Middle: Brand Matrix - Refined Typography */}
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

                    {/* Footer: Infrastructure Status Matrix */}
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

                {/* Right Panel: Clean Authentication Matrix */}
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
    }

    const CustomTooltip = ({ active, payload, label }) => {
        if (active && payload && payload.length) {
            return (
                <div className="bg-white/90 dark:bg-zinc-900/90 backdrop-blur-xl p-4 border border-slate-200 dark:border-white/5 rounded-2xl shadow-2xl shadow-black/20">
                    <p className="text-[10px] font-black text-slate-400 uppercase tracking-widest mb-1">{label}</p>
                    <p className="text-lg font-black text-primary">{payload[0].value} <span className="text-[10px] uppercase">Tasks</span></p>
                </div>
            );
        }
        return null;
    };

    return (
        <div className="flex h-screen bg-[#f8fafc] dark:bg-[#050505] text-[#0f172a] dark:text-[#f8fafc] overflow-hidden font-sans selection:bg-primary selection:text-white">
            <aside className="w-80 bg-white/50 dark:bg-zinc-950/20 backdrop-blur-3xl border-r border-slate-200/50 dark:border-white/5 flex flex-col p-8 space-y-12 z-30 shadow-[4px_0_24px_rgba(0,0,0,0.02)] relative overflow-y-auto scrollbar-hide">
                <div className="absolute top-0 left-0 w-full h-[500px] bg-gradient-to-b from-primary/5 to-transparent pointer-events-none" />

                <div className="flex items-center space-x-4 px-2 relative z-10 group">
                    <div className="p-3.5 premium-gradient rounded-2xl text-white shadow-[0_8px_20px_rgba(59,130,246,0.3)] ring-4 ring-primary/10 group-hover:scale-110 group-hover:rotate-6 transition-all duration-500 cursor-pointer">
                        <Bot size={28} className="animate-pulse-soft" />
                    </div>
                    <div>
                        <h1 className="text-2xl font-[900] text-slate-900 dark:text-white tracking-[-0.05em] uppercase leading-none">Zee<span className="text-primary italic">Mirror</span></h1>
                        <p className="text-[9px] font-black text-primary/60 tracking-[0.4em] uppercase mt-1.5 opacity-80">Cloud Interface</p>
                    </div>
                </div>

                <nav className="flex-1 space-y-2 relative z-10">
                    {[
                        { id: 'overview', label: 'Overview', icon: Activity },
                        { id: 'tasks', label: 'Queued Tasks', icon: Download },
                        { id: 'files', label: 'File Explorer', icon: Folder },
                        { id: 'analytics', label: 'Analytics', icon: BarChart3 },
                        { id: 'logs', label: 'System Logs', icon: Terminal },
                        { id: 'settings', label: 'Environment', icon: Settings },
                    ].map(item => (
                        <button
                            key={item.id}
                            onClick={() => setActiveTab(item.id)}
                            className={`flex items-center space-x-4 w-full px-6 py-4 rounded-[1.75rem] transition-all duration-500 font-black text-xs uppercase tracking-wider group ${activeTab === item.id
                                ? 'bg-primary text-white shadow-[0_12px_24px_-8px_rgba(59,130,246,0.4)] scale-[1.03]'
                                : 'text-slate-400 dark:text-zinc-500 hover:bg-slate-100 dark:hover:bg-white/5 hover:text-primary dark:hover:text-primary transition-all'
                                }`}
                        >
                            <item.icon size={18} className={`${activeTab === item.id ? 'scale-110' : 'group-hover:scale-110 group-hover:text-primary'} transition-all`} />
                            <span>{item.label}</span>
                            {item.id === 'tasks' && tasks.length > 0 && (
                                <span className={`ml-auto px-2 py-0.5 rounded-lg text-[9px] font-black ${activeTab === 'tasks' ? 'bg-white text-primary' : 'bg-primary/10 text-primary animate-pulse'}`}>
                                    {tasks.length}
                                </span>
                            )}
                        </button>
                    ))}
                </nav>

                <div className="mt-auto pt-8 border-t border-slate-200/50 dark:border-white/5 space-y-4 relative z-10">
                    <div className="p-5 bg-slate-50/50 dark:bg-zinc-900/40 rounded-[2rem] flex items-center space-x-4 border border-slate-100 dark:border-white/5 shadow-inner">
                        <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center text-primary shadow-lg"><ShieldCheck size={20} /></div>
                        <div className="flex-1 min-w-0">
                            <p className="text-[9px] font-black uppercase tracking-widest text-slate-800 dark:text-white">Active Node</p>
                            <div className="flex items-center text-[8px] text-green-500 font-black uppercase mt-1"><span className="w-1 h-1 rounded-full bg-green-500 mr-2 animate-ping" />Syncing</div>
                        </div>
                    </div>

                    <button
                        onClick={() => { if (confirm('Disconnect from Cloud Engine?')) { localStorage.clear(); window.location.reload(); } }}
                        className="w-full flex items-center justify-center space-x-3 p-5 rounded-[2rem] bg-red-500/5 hover:bg-red-500 text-red-500 hover:text-white border border-red-500/10 transition-all duration-500 group"
                    >
                        <LogOut size={18} className="group-hover:-translate-x-1 transition-transform" />
                        <span className="text-[10px] font-black uppercase tracking-[0.2em]">Exit Engine</span>
                    </button>
                </div>
            </aside>

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
                    <div className="space-y-16 relative z-10 animate-in fade-in zoom-in-95 duration-1000">
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
                            <StatsCard icon={TrendingUp} label="Processing" value={tasks.length} subLabel="Active Node Latency" color="primary" />
                            <StatsCard icon={Download} label="Deliveries" value={stats.total_tasks || 0} subLabel="Total Batch Success" color="blue" />
                            <StatsCard icon={HardDrive} label="Throughput" value={formatBytes(stats.total_bandwidth || 0)} subLabel="Combined Data Flow" color="indigo" />
                            <StatsCard icon={ShieldCheck} label="Authorized" value={stats.users_count || 1} subLabel="Secure Edge Points" color="green" />
                        </div>

                        <div className="grid grid-cols-1 xl:grid-cols-3 gap-12">
                            <div className="xl:col-span-2 space-y-10">
                                <div className="flex justify-between items-end px-2">
                                    <div className="flex items-center space-x-4">
                                        <div className="p-3 bg-primary/10 rounded-[1.25rem] text-primary"><Clock size={24} /></div>
                                        <h3 className="text-3xl font-black text-slate-900 dark:text-white tracking-tight">Active Processes</h3>
                                    </div>
                                    <button onClick={() => setActiveTab('tasks')} className="px-6 py-2.5 bg-slate-100 dark:bg-zinc-900/60 rounded-xl text-[9px] font-black uppercase tracking-widest text-primary hover:bg-primary hover:text-white transition-all shadow-sm border border-slate-200 dark:border-white/5">
                                        View All Instances
                                    </button>
                                </div>
                                <div className="space-y-6">
                                    {tasks.length > 0 ? (
                                        <div className="grid grid-cols-1 gap-6">
                                            {tasks.slice(0, 3).map(task => (
                                                <TaskRow key={task.id} task={task} formatBytes={formatBytes} onCancel={cancelTask} />
                                            ))}
                                        </div>
                                    ) : (
                                        <div className="glass-card py-32 rounded-[3.5rem] flex flex-col items-center justify-center text-center group border-dashed border-2 border-slate-200 dark:border-white/10">
                                            <div className="p-12 bg-slate-100 dark:bg-white/5 rounded-full mb-8 group-hover:scale-110 transition-all duration-700 shadow-inner">
                                                <Activity size={72} className="text-slate-300 dark:text-zinc-800 opacity-60" />
                                            </div>
                                            <h4 className="text-2xl font-black text-slate-400 dark:text-zinc-600 uppercase tracking-[0.2em] leading-relaxed">Cluster Idle<br /><span className="text-[10px] text-primary/40 tracking-[0.5em]">Waiting for distribution protocols</span></h4>
                                        </div>
                                    )}
                                </div>
                            </div>

                            <div className="space-y-12">
                                <div className="flex items-center space-x-4 px-2">
                                    <div className="p-3 bg-indigo-500/10 rounded-[1.25rem] text-indigo-400"><Layers size={24} /></div>
                                    <h3 className="text-3xl font-black text-slate-900 dark:text-white tracking-tight">Node Metrics</h3>
                                </div>
                                <div className="glass-card p-12 rounded-[4rem] space-y-12 relative overflow-hidden group">
                                    <div className="absolute top-0 right-0 p-8 opacity-[0.03] group-hover:opacity-[0.07] transition-all duration-1000 rotate-12"><Zap size={140} /></div>
                                    <div className="space-y-12 relative z-10">
                                        <HealthBar label="Global CPU Load" value={Math.round(system.cpu || 0)} color="from-primary to-blue-400" />
                                        <HealthBar label="Memory Utilization" value={Math.round(system.ram || 0)} color="from-indigo-500 to-purple-400" />
                                        <HealthBar label="Storage Allocation" value={Math.round(system.disk || 0)} color="from-blue-600 to-cyan-400" />
                                    </div>

                                    <div className="pt-12 border-t border-slate-100 dark:border-white/5 space-y-7 relative z-10">
                                        <StatusItem label="Edge Service" value="Operational" dot="bg-green-500" />
                                        <StatusItem label="Architecture" value={`${system.os || 'Linux'} ${system.arch || 'x64'}`} color="text-slate-500" />
                                        <StatusItem label="Active Uptime" value={system.uptime ? `${Math.floor(system.uptime / 3600)}h ${Math.floor((system.uptime % 3600) / 60)}m` : 'Stable'} color="text-primary" />
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                )}

                {activeTab === 'analytics' && (
                    <div className="animate-in slide-in-from-right-10 duration-700 space-y-12">
                        <div className="grid grid-cols-1 lg:grid-cols-2 gap-12">
                            <div className="glass-card p-12 rounded-[4rem] shadow-2xl space-y-8">
                                <div className="flex items-center justify-between px-2">
                                    <h3 className="text-3xl font-black text-slate-900 dark:text-white">Load Velocity</h3>
                                    <TrendingUp className="text-primary" size={28} />
                                </div>
                                <div className="h-[350px] w-full mt-8" style={{ minHeight: '350px' }}>
                                    {analyticsData.length > 0 ? (
                                        <ResponsiveContainer width="100%" height="100%" minHeight={350}>
                                            <AreaChart data={analyticsData}>
                                                <defs>
                                                    <linearGradient id="colorTasks" x1="0" y1="0" x2="0" y2="1">
                                                        <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                                                        <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                                                    </linearGradient>
                                                </defs>
                                                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke={isDarkMode ? '#ffffff05' : '#00000005'} />
                                                <XAxis
                                                    dataKey="Date"
                                                    axisLine={false}
                                                    tickLine={false}
                                                    tick={{ fill: '#94a3b8', fontSize: 10, fontWeight: 900 }}
                                                    tickFormatter={(str) => new Date(str).toLocaleDateString('en-US', { weekday: 'short' })}
                                                />
                                                <YAxis hide />
                                                <Tooltip content={<CustomTooltip />} cursor={{ stroke: '#3b82f6', strokeWidth: 2 }} />
                                                <Area type="monotone" dataKey="TotalTasks" stroke="#3b82f6" strokeWidth={4} fillOpacity={1} fill="url(#colorTasks)" />
                                            </AreaChart>
                                        </ResponsiveContainer>
                                    ) : (
                                        <div className="flex items-center justify-center h-full text-slate-400">No Data Available</div>
                                    )}
                                </div>
                            </div>

                            <div className="glass-card p-12 rounded-[4rem] shadow-2xl space-y-8">
                                <div className="flex items-center justify-between px-2">
                                    <h3 className="text-3xl font-black text-slate-900 dark:text-white">Distribution Ratio</h3>
                                    <BarChart3 className="text-indigo-400" size={28} />
                                </div>
                                <div className="h-[350px] w-full mt-8" style={{ minHeight: '350px' }}>
                                    {analyticsData.length > 0 ? (
                                        <ResponsiveContainer width="100%" height="100%" minHeight={350}>
                                            <BarChart data={analyticsData}>
                                                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke={isDarkMode ? '#ffffff05' : '#00000005'} />
                                                <XAxis
                                                    dataKey="Date"
                                                    axisLine={false}
                                                    tickLine={false}
                                                    tick={{ fill: '#94a3b8', fontSize: 10, fontWeight: 900 }}
                                                    tickFormatter={(str) => new Date(str).getDate()}
                                                />
                                                <Tooltip content={<CustomTooltip />} cursor={{ fill: isDarkMode ? '#ffffff05' : '#00000005' }} />
                                                <Bar dataKey="TotalTasks" radius={[10, 10, 10, 10]} barSize={32}>
                                                    {analyticsData.map((entry, index) => (
                                                        <Cell key={`cell-${index}`} fill={index === analyticsData.length - 1 ? '#3b82f6' : '#94a3b830'} />
                                                    ))}
                                                </Bar>
                                            </BarChart>
                                        </ResponsiveContainer>
                                    ) : (
                                        <div className="flex items-center justify-center h-full text-slate-400">No Data Available</div>
                                    )}
                                </div>
                            </div>
                        </div>

                        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
                            <AnalyticsStat label="Peak Performance" value={Math.max(...analyticsData.map(d => d.TotalTasks), 0)} unit="Tasks/Day" />
                            <AnalyticsStat label="Weekly Volume" value={analyticsData.reduce((acc, curr) => acc + curr.TotalTasks, 0)} unit="Aggregate" />
                            <AnalyticsStat label="Node Efficiency" value="98.4%" unit="Realtime" />
                        </div>
                    </div>
                )}

                {activeTab === 'files' && (
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
                            <button onClick={() => fetchExplorer(explorerPath)} className="p-4 bg-slate-50 dark:bg-white/5 text-slate-400 hover:text-primary rounded-2xl transition-all shadow-sm border border-slate-100 dark:border-white/5"><RefreshCcw size={24} /></button>
                        </div>

                        <div className="glass-card rounded-[3.5rem] overflow-hidden shadow-2xl border border-slate-100 dark:border-white/5 bg-white/80 dark:bg-zinc-900/40 backdrop-blur-3xl px-2">
                            <div className="grid grid-cols-12 gap-4 px-10 py-8 bg-slate-50/20 dark:bg-white/5 border-b border-slate-100 dark:border-white/5 text-[11px] font-black uppercase tracking-[0.2em] text-slate-400">
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
                                                    <span className="text-[10px] font-black uppercase tracking-widest text-primary/60">Edge Link</span>
                                                    <span className="w-1.5 h-1.5 rounded-full bg-slate-300 dark:bg-white/10" />
                                                    <span className="text-[10px] font-bold text-slate-400 truncate opacity-60 font-mono italic">NODE-ID: {file.name.slice(0, 8).toUpperCase()}</span>
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
                                                    <button onClick={() => getExternalLink(file.name)} className="p-4 text-slate-400 hover:text-primary transition-all rounded-2xl hover:bg-slate-100 dark:hover:bg-white/10 hidden md:flex border border-transparent hover:border-primary/20">
                                                        <ExternalLink size={22} />
                                                    </button>
                                                    <button onClick={() => deleteFile(file.name)} className="p-4 text-red-400 hover:text-white hover:bg-red-500 rounded-2xl transition-all shadow-xl hover:shadow-red-500/30 active:scale-95">
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
                )}

                {activeTab === 'logs' && (
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
                )}

                {activeTab === 'settings' && (
                    <div className="max-w-3xl animate-in slide-in-from-right-10 duration-700">
                        <form onSubmit={handleUpdateSettings} className="glass-card p-16 rounded-[4.5rem] space-y-12 shadow-[0_50px_100px_-20px_rgba(0,0,0,0.1)] border-[#ffffff50] dark:border-[#ffffff05] relative overflow-hidden group">
                            <div className="absolute top-0 right-0 w-64 h-64 bg-primary/5 rounded-full blur-[80px] -mr-32 -mt-32" />

                            <div className="flex items-center space-x-6 mb-4 relative z-10">
                                <div className="p-5 premium-gradient rounded-[2rem] text-white shadow-2xl shadow-primary/30 ring-8 ring-primary/5 group-hover:rotate-12 transition-transform duration-700"><Settings size={36} /></div>
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
                )}
            </main>
        </div>
    );
};

const StatsCard = ({ icon: Icon, label, value, subLabel, color }) => (
    <div className={`glass-card p-10 rounded-[3.5rem] group hover:scale-[1.05] transition-all duration-700 hover:shadow-2xl relative overflow-hidden cursor-default border-t-2 ${color === 'primary' ? 'border-primary/40' : color === 'blue' ? 'border-blue-400/40' : color === 'indigo' ? 'border-indigo-400/40' : 'border-green-400/40'}`}>
        <div className="absolute top-[-40px] right-[-40px] w-48 h-48 premium-gradient rounded-full opacity-[0.03] group-hover:opacity-[0.08] transition-all duration-1000 blur-2xl group-hover:scale-150" />
        <div className={`w-16 h-16 rounded-2xl bg-${color}-500/10 flex items-center justify-center mb-8 text-${color}-500 group-hover:scale-110 group-hover:rotate-6 transition-all duration-700 shadow-inner ring-4 ring-${color}-500/5`}>
            {color === 'primary' ? <Icon size={32} className="text-primary" /> : <Icon size={32} />}
        </div>
        <div className="space-y-1 relative z-10">
            <p className="text-[9px] font-black text-slate-400 dark:text-zinc-500 uppercase tracking-[0.4em] mb-1 group-hover:text-primary transition-colors">{label}</p>
            <p className="text-4xl lg:text-5xl font-black tracking-[-0.05em] text-slate-900 dark:text-white leading-none mb-2">{value}</p>
            <div className="flex items-center space-x-2">
                <div className={`w-3 h-1 rounded-full bg-${color}-500 ${color === 'primary' ? 'bg-primary' : ''} group-hover:w-6 transition-all`} />
                <p className="text-[10px] font-black text-slate-400 dark:text-zinc-400 uppercase tracking-widest opacity-60 transition-opacity group-hover:opacity-100">{subLabel}</p>
            </div>
        </div>
    </div>
);

const TaskRow = ({ task, formatBytes, onCancel }) => (
    <div className="glass-card p-8 rounded-[3rem] flex flex-col space-y-7 hover:scale-[1.02] transition-all duration-500 border-l-[6px] border-l-primary group relative overflow-hidden shadow-xl hover:shadow-primary/5 text-slate-800 dark:text-white">
        <div className="absolute top-0 right-0 p-6 opacity-[0.02] group-hover:opacity-[0.05] transition-all duration-1000 rotate-12 -mr-8 -mt-8">
            <Download size={120} />
        </div>

        <div className="flex items-start justify-between relative z-10 gap-6">
            <div className="flex items-center space-x-6 flex-1 min-w-0">
                <div className="p-4 bg-primary/10 text-primary rounded-[1.5rem] group-hover:premium-gradient group-hover:text-white transition-all duration-700 shadow-inner shrink-0 group-hover:rotate-3">
                    <Download size={28} className={task.status === 'downloading' ? 'animate-bounce' : ''} />
                </div>
                <div className="flex-1 min-w-0">
                    <div className="flex items-center flex-wrap gap-2 mb-2">
                        <span className="px-2.5 py-1 bg-primary text-white text-[9px] font-black uppercase tracking-widest rounded-lg shadow-lg shadow-primary/20">
                            {task.type}
                        </span>
                        <span className="px-2.5 py-1 bg-slate-100 dark:bg-white/5 text-slate-900 dark:text-white text-[8px] font-black uppercase tracking-[0.2em] rounded-lg border border-slate-200 dark:border-white/5">
                            {task.status}
                        </span>
                    </div>
                    <h4 className="font-black text-xl text-slate-900 dark:text-white truncate pr-4 leading-tight group-hover:text-primary transition-colors" title={task.fileName || task.id}>
                        {task.fileName || `Thread: ${task.id}`}
                    </h4>
                </div>
            </div>
            <button
                onClick={() => onCancel(task.id)}
                className="p-4 text-slate-300 hover:text-white hover:bg-red-500 rounded-2xl transition-all shrink-0 active:scale-90 border border-transparent hover:border-red-500/20 shadow-sm"
            >
                <Trash2 size={24} />
            </button>
        </div>

        <div className="space-y-6 relative z-10">
            <div className="flex items-end justify-between px-1">
                <div className="flex items-center space-x-12">
                    <div className="space-y-1">
                        <p className="text-[10px] font-black text-slate-400 uppercase tracking-widest leading-none mb-1">Bandwidth</p>
                        <div className="flex items-center text-sm font-black text-slate-800 dark:text-slate-200">
                            <TrendingUp size={16} className="mr-2 text-primary" /> {formatBytes(task.speed || 0)}<span className="text-[10px] font-bold opacity-60 ml-1">/s</span>
                        </div>
                    </div>
                    <div className="space-y-1">
                        <p className="text-[10px] font-black text-slate-400 uppercase tracking-widest leading-none mb-1">Time Buffer</p>
                        <div className="flex items-center text-sm font-black text-slate-800 dark:text-slate-200">
                            <Clock size={16} className="mr-2 text-indigo-400" /> {task.eta ? `${Math.floor(task.eta / 1e9 / 60)}m ${Math.floor(task.eta / 1e9) % 60}s` : 'Stable'}
                        </div>
                    </div>
                </div>
                <div className="text-right">
                    <span className="text-4xl font-black text-primary tracking-tighter tabular-nums drop-shadow-sm">
                        {(task.progress || 0).toFixed(1)}<span className="text-xs ml-0.5">%</span>
                    </span>
                </div>
            </div>

            <div className="relative group/bar">
                <div className="h-2.5 w-full bg-slate-100 dark:bg-white/5 rounded-full overflow-hidden shadow-inner flex p-0.5">
                    <div
                        className="h-full bg-primary rounded-full transition-all duration-1000 shadow-[0_0_20px_rgba(59,130,246,0.6)] relative overflow-hidden"
                        style={{ width: `${task.progress || 0}%` }}
                    >
                        <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/30 to-transparent -translate-x-full animate-[shimmer_1.5s_infinite]" />
                    </div>
                </div>
                <div className="absolute -top-1 opacity-0 group-hover/bar:opacity-100 transition-opacity right-0 -mr-2 w-1.5 h-4 bg-primary rounded-full blur-sm" />
            </div>
        </div>
    </div>
);

const HealthBar = ({ label, value, color }) => (
    <div className="space-y-4 group">
        <div className="flex justify-between text-[11px] font-[900] text-slate-400 dark:text-zinc-500 uppercase tracking-[0.3em] group-hover:text-primary transition-colors px-1">
            <span>{label}</span>
            <span className="text-slate-900 dark:text-white font-black tracking-widest">{value}<span className="text-[8px] ml-0.5 opacity-40">%</span></span>
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
);

const StatusItem = ({ label, value, color, dot }) => (
    <div className="flex justify-between items-center group/item hover:bg-slate-100 dark:hover:bg-white/[0.03] p-1.5 rounded-xl transition-all px-3">
        <div className="flex items-center space-x-3">
            {dot && <div className={`w-2 h-2 rounded-full ${dot} animate-pulse`} />}
            <span className="text-[11px] font-black text-slate-400 dark:text-zinc-500 uppercase tracking-widest group-hover/item:text-primary transition-colors">{label}</span>
        </div>
        <span className={`text-[11px] font-[900] tracking-tighter uppercase ${color || 'text-slate-900 dark:text-white'}`}>{value}</span>
    </div>
);

const AnalyticsStat = ({ label, value, unit }) => (
    <div className="glass-card p-10 rounded-[3rem] text-center space-y-2 group hover:scale-[1.05] transition-all duration-500 shadow-xl">
        <p className="text-[10px] font-black text-slate-400 dark:text-zinc-500 uppercase tracking-widest">{label}</p>
        <p className="text-5xl font-black text-slate-900 dark:text-white tracking-tighter group-hover:text-primary transition-colors">{value}</p>
        <p className="text-[9px] font-black text-primary/60 uppercase tracking-[0.4em] pt-2">{unit}</p>
    </div>
);

export default Dashboard;
