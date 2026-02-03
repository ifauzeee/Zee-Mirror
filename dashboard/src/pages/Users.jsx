import React, { useState } from 'react';
import { useUsers } from '../hooks/useUsers';
import { formatBytes } from '../utils/format';
import { usePopups } from '../context/PopupContext';

const Users = ({ apiToken }) => {
    const { users, loading, error, updateUser, deleteUser, fetchUsers } = useUsers(apiToken);
    const { showConfirm, showAlert, showToast } = usePopups();
    const [editingUser, setEditingUser] = useState(null);
    const [isModalOpen, setIsModalOpen] = useState(false);

    // Form state
    const [role, setRole] = useState('');
    const [maxTasks, setMaxTasks] = useState(-1);
    const [maxBW, setMaxBW] = useState(-1);
    const [bwStr, setBwStr] = useState('-1');
    const [expiresAt, setExpiresAt] = useState('');

    const handleEdit = (user) => {
        setEditingUser(user);
        setRole(user.role);
        setMaxTasks(user.maxDailyTasks);
        setMaxBW(user.maxDailyBandwidth);
        setBwStr(user.maxDailyBandwidth === -1 ? '-1' : formatBytes(user.maxDailyBandwidth));
        setExpiresAt(user.expiresAt?.Valid ? user.expiresAt.Time.split('T')[0] : '');
        setIsModalOpen(true);
    };

    const handleSave = async (e) => {
        e.preventDefault();
        const result = await updateUser({
            id: editingUser.id,
            role,
            maxDailyTasks: parseInt(maxTasks),
            maxDailyBandwidth: maxBW, // We should probably have a better way to input this, but for now...
            expiresAt: expiresAt ? new Date(expiresAt).toISOString() : ''
        });
        if (result.success) {
            setIsModalOpen(false);
            setEditingUser(null);
            showToast('User settings updated successfully', 'success');
        } else {
            showAlert('Update Failed', result.error, { type: 'error' });
        }
    };

    const handleDelete = async (id) => {
        if (await showConfirm('Delete User', 'Are you sure you want to permanently delete this user? This action cannot be undone.')) {
            const result = await deleteUser(id);
            if (result.success) {
                showToast('User deleted successfully', 'success');
            } else {
                showAlert('Delete Failed', result.error, { type: 'error' });
            }
        }
    };

    return (
        <div className="space-y-6 animate-fade-in">
            <div className="flex justify-between items-center">
                <div>
                    <h1 className="text-3xl font-bold bg-gradient-to-r from-blue-400 to-indigo-500 bg-clip-text text-transparent">
                        User Management
                    </h1>
                    <p className="text-slate-500 dark:text-gray-400 mt-1">Manage bot access, roles, and daily quotas.</p>
                </div>
                <button
                    onClick={fetchUsers}
                    className="p-2 bg-gray-800/50 rounded-lg hover:bg-gray-700 transition-colors"
                    title="Refresh"
                >
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                    </svg>
                </button>
            </div>

            {error && (
                <div className="bg-red-500/10 border border-red-500/20 text-red-500 p-4 rounded-xl flex items-center gap-3">
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    {error}
                </div>
            )}

            <div className="glass-card overflow-hidden">
                <div className="overflow-x-auto">
                    <table className="w-full text-left">
                        <thead>
                            <tr className="border-b border-slate-200 dark:border-white/5 bg-slate-50 dark:bg-white/5">
                                <th className="px-6 py-4 font-semibold text-slate-600 dark:text-gray-300">User</th>
                                <th className="px-6 py-4 font-semibold text-slate-600 dark:text-gray-300">Role</th>
                                <th className="px-6 py-4 font-semibold text-slate-600 dark:text-gray-300">Quotas (Tasks/BW)</th>
                                <th className="px-6 py-4 font-semibold text-slate-600 dark:text-gray-300">Expires At</th>
                                <th className="px-6 py-4 font-semibold text-slate-600 dark:text-gray-300">Created</th>
                                <th className="px-6 py-4 font-semibold text-slate-600 dark:text-gray-300 text-right">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-white/5">
                            {loading && users.length === 0 ? (
                                <tr>
                                    <td colSpan="6" className="px-6 py-12 text-center text-gray-500">
                                        <div className="flex flex-col items-center gap-2">
                                            <div className="w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
                                            <span>Loading users...</span>
                                        </div>
                                    </td>
                                </tr>
                            ) : users.length === 0 ? (
                                <tr>
                                    <td colSpan="6" className="px-6 py-12 text-center text-gray-500">No users found.</td>
                                </tr>
                            ) : (
                                users.map(user => (
                                    <tr key={user.id} className="hover:bg-white/5 transition-colors group">
                                        <td className="px-6 py-4">
                                            <div className="flex items-center gap-3">
                                                <div className="w-10 h-10 rounded-full bg-slate-100 dark:bg-gradient-to-br from-gray-700 to-gray-800 flex items-center justify-center text-blue-500 dark:text-blue-400 font-bold border border-slate-200 dark:border-white/10">
                                                    {user.username?.[0]?.toUpperCase() || 'U'}
                                                </div>
                                                <div>
                                                    <p className="font-extrabold text-slate-900 dark:text-gray-200">{user.username || 'Unknown User'}</p>
                                                    <p className="text-xs text-slate-500 dark:text-gray-500 family-mono">ID: {user.id}</p>
                                                </div>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <span className={`px-3 py-1 rounded-full text-xs font-medium border ${user.role === 'owner' ? 'bg-purple-500/10 border-purple-500/20 text-purple-400' :
                                                user.role === 'admin' ? 'bg-blue-500/10 border-blue-500/20 text-blue-400' :
                                                    user.role === 'authorized' ? 'bg-green-500/10 border-green-500/20 text-green-400' :
                                                        'bg-gray-500/10 border-gray-500/20 text-gray-400'
                                                }`}>
                                                {user.role.toUpperCase()}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4">
                                            <div className="text-sm">
                                                <p className="text-slate-700 dark:text-gray-300">
                                                    <span className="text-slate-500 dark:text-gray-500">Tasks:</span> {user.maxDailyTasks === -1 ? '∞' : user.maxDailyTasks}
                                                </p>
                                                <p className="text-slate-700 dark:text-gray-300">
                                                    <span className="text-slate-500 dark:text-gray-500">BW:</span> {user.maxDailyBandwidth === -1 ? '∞' : formatBytes(user.maxDailyBandwidth)}
                                                </p>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <span className={`text-sm ${!user.isActive ? 'text-red-500 line-through' : 'text-slate-700 dark:text-gray-300'}`}>
                                                {user.expiresAt?.Valid ? new Date(user.expiresAt.Time).toLocaleDateString() : 'Never'}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4 text-sm text-slate-500 dark:text-gray-400 lowercase">
                                            {new Date(user.createdAt).toLocaleDateString()}
                                        </td>
                                        <td className="px-6 py-4 text-right">
                                            <div className="flex justify-end gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                                                <button
                                                    onClick={() => handleEdit(user)}
                                                    className="p-2 hover:bg-blue-500/10 rounded-lg text-blue-500 dark:text-blue-400 transition-colors"
                                                    title="Edit User"
                                                >
                                                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-5M16.5 3.5a2.121 2.121 0 113 3L7 19l-4 1 1-4L16.5 3.5z" />
                                                    </svg>
                                                </button>
                                                {user.role !== 'owner' && (
                                                    <button
                                                        onClick={() => handleDelete(user.id)}
                                                        className="p-2 hover:bg-red-500/10 rounded-lg text-red-500 dark:text-red-400 transition-colors"
                                                        title="Delete User"
                                                    >
                                                        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                                        </svg>
                                                    </button>
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

            {/* Edit Modal */}
            {isModalOpen && (
                <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fade-in">
                    <div className="glass-card w-full max-w-lg overflow-hidden shadow-2xl animate-scale-in">
                        <div className="bg-slate-50 dark:bg-white/5 px-6 py-4 border-b border-slate-200 dark:border-white/5 flex justify-between items-center">
                            <h2 className="text-xl font-black text-slate-900 dark:text-white uppercase tracking-tight">Edit User: {editingUser?.username || editingUser?.id}</h2>
                            <button
                                onClick={() => setIsModalOpen(false)}
                                className="text-slate-400 hover:text-primary transition-colors"
                                title="Close"
                            >
                                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l18 18" />
                                </svg>
                            </button>
                        </div>
                        <form onSubmit={handleSave} className="p-6 space-y-4">
                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-2">
                                    <label className="text-xs font-semibold text-gray-500 uppercase tracking-wider">Role</label>
                                    <select
                                        value={role}
                                        onChange={(e) => setRole(e.target.value)}
                                        disabled={editingUser?.role === 'owner'}
                                        className="w-full bg-slate-100 dark:bg-gray-900 border border-slate-200 dark:border-white/10 rounded-lg px-4 py-2 text-slate-900 dark:text-gray-200 outline-none focus:border-blue-500/50 transition-colors"
                                    >
                                        <option value="user">User</option>
                                        <option value="authorized">Authorized</option>
                                        <option value="admin">Admin</option>
                                        {editingUser?.role === 'owner' && <option value="owner">Owner</option>}
                                    </select>
                                </div>
                                <div className="space-y-2">
                                    <label className="text-xs font-semibold text-gray-500 uppercase tracking-wider">Daily Tasks</label>
                                    <input
                                        type="number"
                                        value={maxTasks}
                                        onChange={(e) => setMaxTasks(e.target.value)}
                                        className="w-full bg-slate-100 dark:bg-gray-900 border border-slate-200 dark:border-white/10 rounded-lg px-4 py-2 text-slate-900 dark:text-gray-200 outline-none focus:border-blue-500/50 transition-colors"
                                    />
                                    <p className="text-[10px] text-slate-500 dark:text-gray-600 font-bold">-1 for unlimited</p>
                                </div>
                            </div>

                            <div className="space-y-2">
                                <label className="text-xs font-semibold text-gray-500 uppercase tracking-wider">Max Daily Bandwidth (bytes)</label>
                                <div className="flex gap-3">
                                    <input
                                        type="number"
                                        value={maxBW}
                                        onChange={(e) => setMaxBW(parseInt(e.target.value))}
                                        className="flex-1 bg-slate-100 dark:bg-gray-900 border border-slate-200 dark:border-white/10 rounded-lg px-4 py-2 text-slate-900 dark:text-gray-200 outline-none focus:border-blue-500/50 transition-colors"
                                    />
                                    <div className="bg-slate-50 dark:bg-white/5 border border-slate-200 dark:border-white/10 px-4 py-2 rounded-lg text-slate-500 dark:text-gray-400 text-sm whitespace-nowrap">
                                        {maxBW === -1 ? 'Unlimited' : formatBytes(maxBW)}
                                    </div>
                                </div>
                            </div>

                            <div className="space-y-2">
                                <label className="text-xs font-semibold text-gray-500 uppercase tracking-wider">Expires At</label>
                                <input
                                    type="date"
                                    value={expiresAt}
                                    onChange={(e) => setExpiresAt(e.target.value)}
                                    className="w-full bg-slate-100 dark:bg-gray-900 border border-slate-200 dark:border-white/10 rounded-lg px-4 py-2 text-slate-900 dark:text-gray-200 outline-none focus:border-blue-500/50 transition-colors"
                                />
                            </div>

                            <div className="pt-4 flex gap-3">
                                <button
                                    type="button"
                                    onClick={() => setIsModalOpen(false)}
                                    className="flex-1 px-4 py-2 rounded-lg bg-slate-200 dark:bg-gray-800 hover:bg-slate-300 dark:hover:bg-gray-700 text-slate-700 dark:text-gray-300 transition-colors font-black uppercase text-xs tracking-widest"
                                >
                                    Cancel
                                </button>
                                <button
                                    type="submit"
                                    className="flex-1 px-4 py-2 rounded-lg bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white transition-all font-medium shadow-lg shadow-blue-500/20"
                                >
                                    Save Changes
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            )}
        </div>
    );
};

export default Users;
