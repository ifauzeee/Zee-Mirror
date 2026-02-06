import React, { useState } from 'react';
import { useUsers } from '../hooks/useUsers';
import { formatBytes } from '../utils/format';
import { usePopups } from '../context/PopupContext';
import { UserPlus, RefreshCw, Edit2, Trash2, X, Shield, Activity, Calendar, Fingerprint, Key, HardDrive, Cpu } from 'lucide-react';

const Users = ({ apiToken }) => {
    const { users, loading, error, updateUser, deleteUser, addUser, fetchUsers } = useUsers(apiToken);
    const { showConfirm, showAlert, showToast } = usePopups();
    const [editingUser, setEditingUser] = useState(null);
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [isAdding, setIsAdding] = useState(false);

    const [role, setRole] = useState('authorized');
    const [maxTasks, setMaxTasks] = useState(-1);
    const [maxBW, setMaxBW] = useState(-1);
    const [expiresAt, setExpiresAt] = useState('');
    const [newID, setNewID] = useState('');
    const [newUsername, setNewUsername] = useState('');

    const handleEdit = (user) => {
        setEditingUser(user);
        setIsAdding(false);
        setRole(user.role);
        setMaxTasks(user.maxDailyTasks);
        setMaxBW(user.maxDailyBandwidth);
        setExpiresAt(user.expiresAt?.Valid ? user.expiresAt.Time.split('T')[0] : '');
        setIsModalOpen(true);
    };

    const handleAddNew = () => {
        setEditingUser(null);
        setIsAdding(true);
        setRole('authorized');
        setMaxTasks(-1);
        setMaxBW(-1);
        setExpiresAt('');
        setNewID('');
        setNewUsername('');
        setIsModalOpen(true);
    };

    const handleSave = async (e) => {
        e.preventDefault();
        const userData = {
            role,
            maxDailyTasks: parseInt(maxTasks),
            maxDailyBandwidth: maxBW,
            expiresAt: expiresAt ? new Date(expiresAt).toISOString() : ''
        };

        let result;
        if (isAdding) {
            if (!newID) {
                showAlert('Identity Required', 'Telegram User ID is necessary.', { type: 'error' });
                return;
            }
            result = await addUser({
                id: parseInt(newID),
                username: newUsername,
                ...userData
            });
        } else {
            result = await updateUser({
                id: editingUser.id,
                ...userData
            });
        }

        if (result.success) {
            setIsModalOpen(false);
            setEditingUser(null);
            showToast(isAdding ? 'Subject authorized' : 'Access updated', 'success');
        } else {
            showAlert('Operation Failed', result.error, { type: 'error' });
        }
    };

    const handleDelete = async (id) => {
        if (await showConfirm('Revoke Access', 'Permanently disconnect this subject?')) {
            const result = await deleteUser(id);
            if (result.success) {
                showToast('Link terminated', 'success');
            } else {
                showAlert('Revoke Failed', result.error, { type: 'error' });
            }
        }
    };

    return (
        <div className="max-w-7xl mx-auto space-y-8 pb-20 animate-in fade-in slide-in-from-bottom-4 duration-700">
            {/* Minimalist Header */}
            <div className="flex flex-col md:flex-row justify-between items-center gap-6 px-4">
                <div className="text-center md:text-left">
                    <div className="flex items-center justify-center md:justify-start gap-3 mb-2">
                        <div className="w-2 h-2 rounded-full bg-primary animate-pulse shadow-[0_0_10px_rgba(59,130,246,0.8)]" />
                        <span className="text-[10px] font-black uppercase tracking-[0.4em] text-primary/80">Security Protocol</span>
                    </div>
                    <h1 className="text-5xl font-black text-slate-900 dark:text-white tracking-tighter mb-2">
                        Subject Matrix
                    </h1>
                    <p className="text-sm font-medium text-slate-500 dark:text-slate-400">
                        Manage neural link authorizations and resource allocations.
                    </p>
                </div>

                <div className="flex items-center gap-3">
                    <button
                        onClick={handleAddNew}
                        className="px-8 py-4 bg-primary text-white rounded-2xl flex items-center gap-3 font-bold text-xs uppercase tracking-widest hover:brightness-110 active:scale-95 transition-all shadow-lg shadow-primary/20 group"
                    >
                        <UserPlus size={18} className="group-hover:rotate-12 transition-transform" />
                        Authorize New
                    </button>
                    <button
                        onClick={fetchUsers}
                        className={`p-4 bg-white dark:bg-zinc-900 border border-slate-200 dark:border-white/5 rounded-2xl hover:text-primary transition-all shadow-md ${loading ? 'opacity-50 pointer-events-none' : ''}`}
                    >
                        <RefreshCw size={20} className={loading ? 'animate-spin' : ''} />
                    </button>
                </div>
            </div>

            {error && (
                <div className="mx-4 bg-red-500/10 border border-red-500/20 text-red-500 p-4 rounded-2xl flex items-center gap-4 text-xs font-bold">
                    <Activity size={18} />
                    {error}
                </div>
            )}

            {/* Clean Material Table */}
            <div className="bg-white/80 dark:bg-zinc-900/60 backdrop-blur-xl rounded-3xl border border-slate-200 dark:border-white/5 shadow-2xl shadow-black/5 overflow-hidden mx-4">
                <div className="overflow-x-auto scrollbar-hide">
                    <table className="w-full text-left">
                        <thead>
                            <tr className="border-b border-slate-100 dark:border-white/5">
                                <th className="px-8 py-6 text-[10px] font-black uppercase tracking-widest text-slate-400">Subject</th>
                                <th className="px-6 py-6 text-[10px] font-black uppercase tracking-widest text-slate-400 text-center">Auth Status</th>
                                <th className="px-8 py-6 text-[10px] font-black uppercase tracking-widest text-slate-400 text-center">Load Limits</th>
                                <th className="px-6 py-6 text-[10px] font-black uppercase tracking-widest text-slate-400">Expiry</th>
                                <th className="px-8 py-6 text-[10px] font-black uppercase tracking-widest text-slate-400 text-right">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100 dark:divide-white/5">
                            {loading && users.length === 0 ? (
                                <tr>
                                    <td colSpan="5" className="px-8 py-24 text-center">
                                        <div className="flex flex-col items-center gap-4 opacity-40">
                                            <div className="w-10 h-10 border-4 border-primary border-t-transparent rounded-full animate-spin" />
                                            <span className="text-[10px] font-black uppercase tracking-widest">Syncing Matrix...</span>
                                        </div>
                                    </td>
                                </tr>
                            ) : users.length === 0 ? (
                                <tr>
                                    <td colSpan="5" className="px-8 py-20 text-center text-slate-400 font-bold uppercase text-[10px] tracking-widest italic opacity-50">Empty Database</td>
                                </tr>
                            ) : (
                                users.map((user) => (
                                    <tr key={user.id} className="hover:bg-slate-50 dark:hover:bg-white/[0.02] transition-colors group">
                                        <td className="px-8 py-6">
                                            <div className="flex items-center gap-4">
                                                <div className="w-12 h-12 rounded-xl premium-gradient flex items-center justify-center text-white text-lg font-black shadow-lg">
                                                    {user.username?.[0]?.toUpperCase() || 'U'}
                                                </div>
                                                <div>
                                                    <p className="font-bold text-slate-900 dark:text-white group-hover:text-primary transition-colors">{user.username || 'Anonymous'}</p>
                                                    <p className="text-[10px] font-mono text-slate-400">ID:{user.id}</p>
                                                </div>
                                            </div>
                                        </td>
                                        <td className="px-6 py-6">
                                            <div className="flex justify-center">
                                                <span className={`px-3 py-1 rounded-full text-[10px] font-black uppercase tracking-tighter border ${user.role === 'owner' ? 'bg-purple-500/10 border-purple-500/20 text-purple-600' :
                                                        user.role === 'admin' ? 'bg-blue-500/10 border-blue-500/20 text-blue-600' :
                                                            'bg-emerald-500/10 border-emerald-500/20 text-emerald-600'
                                                    }`}>
                                                    {user.role}
                                                </span>
                                            </div>
                                        </td>
                                        <td className="px-8 py-6">
                                            <div className="flex flex-col items-center gap-1">
                                                <span className="text-[10px] font-bold text-slate-600 dark:text-slate-300">Tasks: {user.maxDailyTasks === -1 ? '∞' : user.maxDailyTasks}</span>
                                                <span className="text-[10px] font-bold text-slate-400">BW: {user.maxDailyBandwidth === -1 ? '∞' : formatBytes(user.maxDailyBandwidth)}</span>
                                            </div>
                                        </td>
                                        <td className="px-6 py-6">
                                            <div className="flex items-center gap-2 text-[10px] font-bold text-slate-500 dark:text-slate-400">
                                                <Calendar size={12} className="text-primary/60" />
                                                {user.expiresAt?.Valid ? new Date(user.expiresAt.Time).toLocaleDateString() : 'INFINITE'}
                                            </div>
                                        </td>
                                        <td className="px-8 py-6 text-right">
                                            <div className="flex justify-end gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                                                <button onClick={() => handleEdit(user)} className="p-3 bg-blue-500/10 text-blue-500 rounded-xl hover:bg-blue-500 hover:text-white transition-all"><Edit2 size={16} /></button>
                                                {user.role !== 'owner' && (
                                                    <button onClick={() => handleDelete(user.id)} className="p-3 bg-red-500/10 text-red-500 rounded-xl hover:bg-red-500 hover:text-white transition-all"><Trash2 size={16} /></button>
                                                )}
                                            </div>
                                        </td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            </div>

            {/* Refined Compact Modal */}
            {isModalOpen && (
                <div className="fixed inset-0 z-50 flex items-center justify-center p-6 bg-slate-900/40 dark:bg-black/80 backdrop-blur-md animate-in fade-in duration-300">
                    <div className="bg-white dark:bg-zinc-900 w-full max-w-xl rounded-3xl overflow-hidden shadow-[0_30px_70px_rgba(0,0,0,0.4)] border border-white/20 dark:border-white/5 animate-in zoom-in-95 duration-300">
                        <div className="px-8 py-6 bg-slate-50 dark:bg-white/[0.03] border-b border-slate-100 dark:border-white/5 flex justify-between items-center">
                            <div>
                                <h2 className="text-xl font-black text-slate-900 dark:text-white tracking-tight uppercase">{isAdding ? 'Authorize Subject' : 'Subject Protocol'}</h2>
                                <p className="text-[10px] font-bold text-primary uppercase tracking-[0.3em] mt-1">{isAdding ? 'New Node Request' : `ID: ${editingUser?.id}`}</p>
                            </div>
                            <button onClick={() => setIsModalOpen(false)} className="p-3 text-slate-400 hover:text-red-500 transition-colors"><X size={20} /></button>
                        </div>

                        <form onSubmit={handleSave} className="p-8 space-y-6">
                            {isAdding && (
                                <div className="grid grid-cols-2 gap-4">
                                    <div className="space-y-2">
                                        <label className="text-[10px] font-black text-slate-400 uppercase tracking-widest px-1">Telegram ID</label>
                                        <input
                                            required type="number" value={newID} onChange={(e) => setNewID(e.target.value)}
                                            className="w-full bg-slate-50 dark:bg-black/20 border border-slate-200 dark:border-white/5 rounded-xl px-4 py-3 text-sm focus:border-primary transition-all font-mono"
                                        />
                                    </div>
                                    <div className="space-y-2">
                                        <label className="text-[10px] font-black text-slate-400 uppercase tracking-widest px-1">Alias (Optional)</label>
                                        <input
                                            type="text" value={newUsername} onChange={(e) => setNewUsername(e.target.value)}
                                            className="w-full bg-slate-50 dark:bg-black/20 border border-slate-200 dark:border-white/5 rounded-xl px-4 py-3 text-sm focus:border-primary transition-all"
                                        />
                                    </div>
                                </div>
                            )}

                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-2">
                                    <label className="text-[10px] font-black text-slate-400 uppercase tracking-widest px-1">Privilege</label>
                                    <select
                                        value={role} onChange={(e) => setRole(e.target.value)}
                                        disabled={!isAdding && editingUser?.role === 'owner'}
                                        className="w-full bg-slate-50 dark:bg-black/20 border border-slate-200 dark:border-white/5 rounded-xl px-4 py-3 text-xs font-bold uppercase transition-all"
                                    >
                                        <option value="user">User</option>
                                        <option value="authorized">Authorized</option>
                                        <option value="admin">Admin</option>
                                    </select>
                                </div>
                                <div className="space-y-2">
                                    <label className="text-[10px] font-black text-slate-400 uppercase tracking-widest px-1">Task Limit</label>
                                    <div className="relative">
                                        <input
                                            type="number" value={maxTasks} onChange={(e) => setMaxTasks(e.target.value)}
                                            className="w-full bg-slate-50 dark:bg-black/20 border border-slate-200 dark:border-white/5 rounded-xl px-4 py-3 text-sm font-bold"
                                        />
                                        <div className="absolute right-3 top-1/2 -translate-y-1/2">
                                            {(maxTasks === '-1' || maxTasks === -1) ? (
                                                <span className="text-[8px] font-black text-emerald-500 bg-emerald-500/10 px-1.5 py-0.5 rounded uppercase">Unlocked</span>
                                            ) : (
                                                <span className="text-[8px] font-black text-slate-400 uppercase">Tasks</span>
                                            )}
                                        </div>
                                    </div>
                                </div>
                            </div>

                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-2">
                                    <label className="text-[10px] font-black text-slate-400 uppercase tracking-widest px-1">Bandwidth Limit</label>
                                    <div className="relative">
                                        <input
                                            type="number" value={maxBW} onChange={(e) => setMaxBW(parseInt(e.target.value))}
                                            className="w-full bg-slate-50 dark:bg-black/20 border border-slate-200 dark:border-white/5 rounded-xl px-4 py-3 text-sm font-bold"
                                        />
                                        <div className="absolute right-3 top-1/2 -translate-y-1/2">
                                            {maxBW === -1 ? (
                                                <span className="text-[8px] font-black text-blue-500 bg-blue-500/10 px-1.5 py-0.5 rounded uppercase">∞</span>
                                            ) : (
                                                <span className="text-[8px] font-black text-slate-400 uppercase">{formatBytes(maxBW)}</span>
                                            )}
                                        </div>
                                    </div>
                                </div>
                                <div className="space-y-2">
                                    <label className="text-[10px] font-black text-slate-400 uppercase tracking-widest px-1">Link Expiry</label>
                                    <input
                                        type="date" value={expiresAt} onChange={(e) => setExpiresAt(e.target.value)}
                                        className="w-full bg-slate-50 dark:bg-black/20 border border-slate-200 dark:border-white/5 rounded-xl px-4 py-3 text-xs font-bold"
                                    />
                                </div>
                            </div>

                            <button
                                type="submit"
                                className="w-full py-4 bg-primary text-white rounded-xl font-black uppercase text-[10px] tracking-[0.3em] shadow-lg shadow-primary/20 hover:brightness-110 transition-all mt-4"
                            >
                                {isAdding ? 'Establish Source Connection' : 'Apply Modification'}
                            </button>
                        </form>
                    </div>
                </div>
            )}
        </div>
    );
};

export default Users;
