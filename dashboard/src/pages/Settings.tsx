import { useState, useEffect, useCallback } from 'react'
import {
  Settings as SettingsIcon,
  Save,
  Terminal,
  Cpu,
  Database,
  RefreshCw,
  Zap,
} from 'lucide-react'
import axios from 'axios'
import { usePopups } from '../hooks/usePopups'

interface SettingsState {
  AutoDeleteMessages: boolean
  DefaultMode: string
  [key: string]: any
}

interface SettingsProps {
  token: string
  initialSettings?: SettingsState
}

const Settings: React.FC<SettingsProps> = ({ token, initialSettings }) => {
  const { showAlert, showToast } = usePopups()
  const [settings, setSettings] = useState<SettingsState>(
    initialSettings || { AutoDeleteMessages: false, DefaultMode: 'mirror' },
  )
  const [rawConfig, setRawConfig] = useState<string>('')
  const [isConfigLoading, setIsConfigLoading] = useState<boolean>(true)
  const [updating, setUpdating] = useState(false)
  const [updateResult, setUpdateResult] = useState<string | null>(null)

  const fetchRawConfig = useCallback(async () => {
    if (!token) return
    try {
      const config = { headers: { 'X-API-Key': token } }
      const res = await axios.get('/api/config', config)
      setRawConfig(res.data.config)
    } catch (err) {
      console.error('Failed to fetch raw config:', err)
    } finally {
      setIsConfigLoading(false)
    }
  }, [token])

  useEffect(() => {
    if (initialSettings) setSettings(initialSettings)
    fetchRawConfig()
  }, [initialSettings, fetchRawConfig])

  const handleUpdateSettings = async () => {
    try {
      const config = { headers: { 'X-API-Key': token } }
      await axios.post('/api/settings', settings, config)
      showToast('Logic configuration synchronized', 'success')
    } catch {
      showAlert('Core Error', 'Failed to synchronize system settings.', { type: 'error' })
    }
  }

  const handleUpdateRawConfig = async () => {
    try {
      const config = { headers: { 'X-API-Key': token } }
      const res = await axios.post('/api/config', { config: rawConfig }, config)
      showToast(res.data.status || 'Manifest updated successfully', 'success')
    } catch {
      showAlert('Manifest Error', 'Failed to update system manifest.', { type: 'error' })
    }
  }

  const handleUpdateTools = async () => {
    setUpdating(true)
    setUpdateResult(null)
    try {
      const config = { headers: { 'X-API-Key': token } }
      const res = await axios.post('/api/tools/update', {}, config)
      setUpdateResult(res.data.output || res.data.error || 'Updated')
    } catch (err: any) {
      setUpdateResult(err.response?.data?.error || err.message || 'Update failed')
    } finally {
      setUpdating(false)
    }
  }

  return (
    <div className="max-w-6xl mx-auto pb-24 animate-in fade-in slide-in-from-bottom-4 duration-1000">
      {/* --- HEADER SECTION --- */}
      <div className="flex flex-col md:flex-row items-start md:items-center justify-between mb-12 gap-6 bg-white/40 dark:bg-white/5 p-10 rounded-[3rem] border border-white/20 dark:border-white/5 backdrop-blur-md">
        <div className="flex items-center space-x-6">
          <div className="p-5 premium-gradient rounded-[2rem] text-white shadow-2xl shadow-primary/30">
            <SettingsIcon size={32} />
          </div>
          <div>
            <h2 className="text-4xl font-black tracking-tighter text-slate-900 dark:text-white">
              Core Control
            </h2>
            <p className="text-[10px] uppercase font-black tracking-[0.3em] text-primary/60 mt-1">
              Zee-Mirror System Administration
            </p>
          </div>
        </div>
        <div className="flex items-center space-x-4">
          <div className="px-6 py-3 bg-white/80 dark:bg-zinc-800 rounded-2xl border border-slate-200 dark:border-white/5 shadow-sm flex items-center space-x-3">
            <div className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
            <span className="text-xs font-black uppercase tracking-widest text-slate-500 dark:text-slate-400">
              Node Status: Active
            </span>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-10">
        {/* --- LEFT COLUMN: BOT LOGIC --- */}
        <div className="lg:col-span-7 space-y-10">
          {/* Bot Behavior Card */}
          <div className="glass-card p-10 rounded-[3.5rem] relative overflow-hidden group">
            <div className="flex items-center space-x-4 mb-10">
              <div className="p-3 bg-primary/10 text-primary rounded-2xl">
                <Zap size={24} />
              </div>
              <h3 className="text-2xl font-black tracking-tight text-slate-900 dark:text-white uppercase">
                Bot Behavior
              </h3>
            </div>

            <div className="space-y-6">
              <div className="flex items-center justify-between p-7 rounded-[2.5rem] bg-slate-50/50 dark:bg-white/5 border border-slate-100 dark:border-white/5 hover:border-primary/30 transition-all duration-500">
                <div className="flex-1 pr-8">
                  <h4 className="font-bold text-slate-900 dark:text-white">
                    Auto-Cleanup Protocol
                  </h4>
                  <p className="text-xs text-slate-500 dark:text-slate-400 mt-1 leading-relaxed">
                    Automatically purge status messages from Telegram once a task completes.
                  </p>
                </div>
                <button
                  onClick={() =>
                    setSettings({ ...settings, AutoDeleteMessages: !settings.AutoDeleteMessages })
                  }
                  className={`w-16 h-8 rounded-full transition-all duration-500 relative p-1 ${settings.AutoDeleteMessages ? 'bg-primary' : 'bg-slate-200 dark:bg-zinc-800'}`}
                >
                  <div
                    className={`w-6 h-6 bg-white rounded-full transition-all duration-300 shadow-lg ${settings.AutoDeleteMessages ? 'translate-x-8' : 'translate-x-0'}`}
                  />
                </button>
              </div>

              <div className="p-7 rounded-[2.5rem] bg-slate-50/50 dark:bg-white/5 border border-slate-100 dark:border-white/5">
                <div className="flex items-center space-x-3 mb-6">
                  <h4 className="font-bold text-slate-900 dark:text-white">
                    Default Transmission Mode
                  </h4>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  {['mirror', 'leech'].map((mode) => (
                    <button
                      key={mode}
                      onClick={() => setSettings({ ...settings, DefaultMode: mode })}
                      className={`py-6 rounded-[2rem] border-2 transition-all duration-500 text-center uppercase tracking-widest text-[10px] font-black ${settings.DefaultMode === mode ? 'border-primary bg-primary text-white shadow-lg shadow-primary/20' : 'border-slate-100 dark:border-white/5 bg-white/5 text-slate-500 dark:text-slate-400 hover:border-primary/20'}`}
                    >
                      {mode}
                    </button>
                  ))}
                </div>
              </div>
            </div>

            <button
              onClick={handleUpdateSettings}
              className="w-full mt-10 bg-slate-900 dark:bg-white text-white dark:text-slate-900 py-6 rounded-[2.5rem] font-black text-sm uppercase tracking-widest flex items-center justify-center space-x-3 hover:scale-[0.98] active:scale-[0.95] transition-all shadow-xl"
            >
              <Save size={20} />
              <span>Commit Logic Sync</span>
            </button>
          </div>

          {/* Quick Stats Card */}
          <div className="grid grid-cols-2 gap-8">
            <div className="glass-card p-10 rounded-[3rem] border-emerald-500/10 dark:border-emerald-500/5">
              <div className="p-3 w-fit bg-emerald-500/10 text-emerald-500 rounded-2xl mb-6">
                <Cpu size={24} />
              </div>
              <p className="text-[10px] font-black uppercase tracking-[0.2em] text-slate-500 mb-1">
                Architecture
              </p>
              <p className="text-xl font-black text-slate-900 dark:text-white uppercase tracking-widest">
                Go 1.25.7 Pure
              </p>
            </div>
            <div className="glass-card p-10 rounded-[3rem] border-blue-500/10 dark:border-blue-500/5">
              <div className="p-3 w-fit bg-blue-500/10 text-blue-500 rounded-2xl mb-6">
                <Database size={24} />
              </div>
              <p className="text-[10px] font-black uppercase tracking-[0.2em] text-slate-500 mb-1">
                Data Matrix
              </p>
              <p className="text-xl font-black text-slate-900 dark:text-white uppercase tracking-widest">
                SQLite CGO-Free
              </p>
            </div>
          </div>

          {/* Tools Card */}
          <div className="glass-card p-10 rounded-[3.5rem] relative overflow-hidden group">
            <div className="flex items-center space-x-4 mb-10">
              <div className="p-3 bg-amber-500/10 text-amber-500 rounded-2xl">
                <Terminal size={24} />
              </div>
              <h3 className="text-2xl font-black tracking-tight text-slate-900 dark:text-white uppercase">
                Tools
              </h3>
            </div>
            <div className="space-y-6">
              <p className="text-sm text-slate-500 dark:text-slate-400 leading-relaxed">
                Update yt-dlp to the latest version using the system package manager.
              </p>
              <button
                onClick={handleUpdateTools}
                disabled={updating}
                className="w-full bg-slate-900 dark:bg-white text-white dark:text-slate-900 py-6 rounded-[2.5rem] font-black text-sm uppercase tracking-widest flex items-center justify-center space-x-3 hover:scale-[0.98] active:scale-[0.95] transition-all shadow-xl disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <RefreshCw size={20} className={updating ? 'animate-spin' : ''} />
                <span>{updating ? 'Updating...' : 'Update yt-dlp'}</span>
              </button>
              {updateResult && (
                <pre className="mt-4 p-6 bg-slate-900/90 dark:bg-black/40 text-emerald-400 font-mono text-[13px] rounded-[2rem] border border-white/5 shadow-inner leading-relaxed whitespace-pre-wrap overflow-x-auto">
                  {updateResult}
                </pre>
              )}
            </div>
          </div>
        </div>

        {/* --- RIGHT COLUMN: MANIFEST EDITOR --- */}
        <div className="lg:col-span-5">
          <div className="glass-card p-12 rounded-[4rem] h-full flex flex-col relative overflow-hidden">
            <div className="absolute top-0 right-0 w-40 h-40 bg-indigo-500/5 rounded-full blur-[60px] -mr-20 -mt-20" />

            <div className="flex items-center justify-between mb-10 relative z-10">
              <div className="flex items-center space-x-4">
                <div className="p-3 bg-indigo-500/10 text-indigo-500 rounded-2xl">
                  <Terminal size={24} />
                </div>
                <h3 className="text-2xl font-black tracking-tight text-slate-900 dark:text-white uppercase">
                  Manifest
                </h3>
              </div>
              <button
                onClick={fetchRawConfig}
                className="p-3 hover:bg-slate-100 dark:hover:bg-white/5 rounded-xl transition-all text-slate-500"
              >
                <RefreshCw size={20} className={isConfigLoading ? 'animate-spin' : ''} />
              </button>
            </div>

            {isConfigLoading ? (
              <div className="flex-1 flex flex-col items-center justify-center space-y-4 py-32 opacity-30 italic font-black uppercase tracking-widest text-xs">
                <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
                <span>Decrypting...</span>
              </div>
            ) : (
              <div className="flex-1 flex flex-col space-y-8">
                <div className="relative group flex-1 min-h-[450px]">
                  <textarea
                    value={rawConfig}
                    onChange={(e) => setRawConfig(e.target.value)}
                    className="w-full h-full bg-slate-900/90 dark:bg-black/40 text-emerald-400 font-mono text-[13px] p-8 rounded-[2.5rem] border border-white/5 outline-none focus:border-indigo-500/30 transition-all shadow-inner leading-relaxed resize-none scrollbar-hide"
                    spellCheck={false}
                  />
                  <div className="absolute bottom-6 right-6 px-3 py-1 bg-white/5 backdrop-blur-md border border-white/10 rounded-full text-[9px] text-slate-500 font-black tracking-widest uppercase">
                    READ_WRITE_OK
                  </div>
                </div>

                <button
                  onClick={handleUpdateRawConfig}
                  className="w-full bg-indigo-600 text-white py-6 rounded-[2.5rem] font-black text-sm uppercase tracking-widest flex items-center justify-center space-x-3 hover:bg-indigo-500 hover:scale-[0.98] transition-all shadow-2xl shadow-indigo-500/20"
                >
                  <Save size={20} />
                  <span>Sync Manifest</span>
                </button>

                <p className="text-center text-[9px] font-black text-slate-400 opacity-40 uppercase tracking-[0.2em] leading-relaxed px-6">
                  System notice: Some environment variables may require a node restart to initialize
                  the new cluster parameters.
                </p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export default Settings
