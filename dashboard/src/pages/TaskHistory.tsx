import { useState, useEffect } from 'react';
import axios from 'axios';
import { History, ChevronLeft, ChevronRight } from 'lucide-react';

interface HistoryTask {
  id: string;
  type: string;
  url: string;
  file_name: string;
  file_size: number;
  status: string;
  md5: string;
  created_at: string;
  updated_at: string;
  error_message: string;
}

interface PaginatedResponse {
  tasks: HistoryTask[];
  total: number;
  page: number;
  limit: number;
}

const TaskHistory: React.FC<{ token: string }> = ({ token }) => {
  const [tasks, setTasks] = useState<HistoryTask[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(0);
  const [statusFilter, setStatusFilter] = useState('');
  const limit = 20;

  useEffect(() => {
    fetchTasks();
  }, [page, statusFilter]);

  const fetchTasks = async () => {
    setLoading(true);
    try {
      const params: Record<string, string | number> = { limit, offset: page * limit };
      if (statusFilter) params.status = statusFilter;
      const res = await axios.get('/api/tasks/history', { params, headers: { 'X-API-Key': token } });
      setTasks(res.data.tasks || res.data || []);
    } catch (err) {
      console.error('Failed to fetch tasks', err);
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (d: string) => {
    if (!d) return '-';
    return new Date(d).toLocaleString();
  };

  const formatSize = (bytes: number) => {
    if (!bytes) return '-';
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${sizes[i]}`;
  };

  const statusColor = (s: string) => {
    switch (s) {
      case 'completed': return 'text-green-600 dark:text-green-400';
      case 'failed': return 'text-red-600 dark:text-red-400';
      case 'downloading': return 'text-blue-600 dark:text-blue-400';
      default: return 'text-gray-600 dark:text-gray-400';
    }
  };

  return (
    <div className="animate-in slide-in-from-right-10 duration-700 space-y-10">
      <div className="flex items-center justify-between px-2">
        <div className="flex items-center space-x-6">
          <div className="p-4 bg-zinc-900 rounded-[1.5rem] text-zinc-400">
            <History size={32} />
          </div>
          <div>
            <h3 className="text-4xl font-black text-slate-900 dark:text-white">Task History</h3>
            <p className="text-[10px] font-black uppercase tracking-[0.3em] text-primary mt-1">
              Completed & Failed Task Archive
            </p>
          </div>
        </div>
      </div>

      <div className="flex items-center space-x-3 overflow-x-auto pb-2 md:pb-0 scrollbar-hide">
        {['', 'completed', 'failed', 'downloading', 'pending', 'queued'].map((s) => (
          <button
            key={s}
            onClick={() => { setStatusFilter(s); setPage(0); }}
            className={`px-6 py-3 rounded-2xl text-[10px] font-black uppercase tracking-widest transition-all ${statusFilter === s
              ? 'bg-primary text-white shadow-lg shadow-primary/20 scale-105'
              : 'bg-white dark:bg-zinc-900/60 text-slate-400 border border-slate-200 dark:border-white/5 hover:bg-slate-50'
              }`}
          >
            {s || 'All'}
          </button>
        ))}
      </div>

      <div className="bg-slate-50 dark:bg-[#0c0c0c] rounded-[3rem] p-10 shadow-[0_40px_100px_-20px_rgba(0,0,0,0.1)] dark:shadow-[0_40px_100px_-20px_rgba(0,0,0,0.4)] border border-slate-200 dark:border-white/5 overflow-hidden relative group">
        <div className="absolute top-0 left-0 w-full h-2 bg-gradient-to-r from-primary via-indigo-600 to-primary opacity-30 group-hover:opacity-100 transition-opacity duration-1000" />

        {loading ? (
          <div className="py-32 flex items-center justify-center">
            <div className="text-slate-400 text-sm font-bold uppercase tracking-widest">Loading...</div>
          </div>
        ) : tasks.length === 0 ? (
          <div className="glass-card py-32 rounded-[3.5rem] flex flex-col items-center justify-center text-center opacity-40">
            <div className="p-8 bg-slate-100 dark:bg-white/5 rounded-full mb-6">
              <History size={48} className="text-slate-400" />
            </div>
            <h4 className="text-xl font-black uppercase tracking-widest text-slate-400">No tasks found</h4>
          </div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full text-left">
                <thead>
                  <tr className="text-[10px] font-black uppercase tracking-widest text-slate-400 dark:text-zinc-500 border-b border-slate-200 dark:border-white/5">
                    <th className="p-4 pl-0">Type</th>
                    <th className="p-4">File</th>
                    <th className="p-4">Status</th>
                    <th className="p-4">Size</th>
                    <th className="p-4">MD5</th>
                    <th className="p-4 pr-0">Date</th>
                  </tr>
                </thead>
                <tbody>
                  {tasks.map((t) => (
                    <tr key={t.id} className="border-b border-slate-100 dark:border-white/[0.03] hover:bg-slate-100/50 dark:hover:bg-white/[0.02] transition-colors">
                      <td className="p-4 pl-0">
                        <span className="px-3 py-1.5 bg-primary/10 text-primary text-[9px] font-black rounded-lg uppercase tracking-widest">
                          {t.type}
                        </span>
                      </td>
                      <td className="p-4">
                        <div className="text-sm font-bold text-slate-700 dark:text-zinc-300 truncate max-w-xs" title={t.file_name || t.url}>
                          {t.file_name || t.url}
                        </div>
                      </td>
                      <td className={`p-4 text-sm font-black uppercase ${statusColor(t.status)}`}>{t.status}</td>
                      <td className="p-4 text-sm font-mono text-slate-500 dark:text-zinc-400">{formatSize(t.file_size)}</td>
                      <td className="p-4">
                        {t.md5 ? (
                          <span className="text-[11px] font-mono text-slate-400 dark:text-zinc-500" title={t.md5}>
                            {t.md5.substring(0, 8)}...
                          </span>
                        ) : (
                          <span className="text-slate-300 dark:text-zinc-700">-</span>
                        )}
                      </td>
                      <td className="p-4 pr-0 text-sm text-slate-500 dark:text-zinc-400 whitespace-nowrap">{formatDate(t.created_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="mt-8 flex items-center justify-center space-x-4">
              <button
                onClick={() => setPage(p => Math.max(0, p - 1))}
                disabled={page === 0}
                className="p-4 bg-white dark:bg-zinc-900/60 border border-slate-200 dark:border-white/5 rounded-2xl disabled:opacity-30 hover:scale-105 active:scale-95 transition-all"
              >
                <ChevronLeft size={20} className="text-slate-500" />
              </button>
              <span className="px-6 py-3 text-sm font-black text-slate-500 dark:text-zinc-400">
                Page {page + 1}
              </span>
              <button
                onClick={() => setPage(p => p + 1)}
                disabled={tasks.length < limit}
                className="p-4 bg-white dark:bg-zinc-900/60 border border-slate-200 dark:border-white/5 rounded-2xl disabled:opacity-30 hover:scale-105 active:scale-95 transition-all"
              >
                <ChevronRight size={20} className="text-slate-500" />
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
};

export default TaskHistory;
