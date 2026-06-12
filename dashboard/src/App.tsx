import { useState, useEffect } from 'react'
import { createBrowserRouter, RouterProvider, Navigate } from 'react-router-dom'
import axios from 'axios'
import DashboardLayout from './layouts/DashboardLayout'
import Login from './pages/Login'
import Overview from './pages/Overview'

import Explorer from './pages/Explorer'
import Analytics from './pages/Analytics'
import Logs from './pages/Logs'
import TorrentSelect from './pages/TorrentSelect'
import Users from './pages/Users'
import Settings from './pages/Settings'
import TaskHistory from './pages/TaskHistory'
import TaskRow from './components/Task/TaskRow'
import useTasks from './hooks/useTasks'
import useSystemStats from './hooks/useSystemStats'
import { usePopups } from './hooks/usePopups'
import { Task } from './types'
import { Activity } from 'lucide-react'


const TasksPage = ({
  tasks,
  cancelTask,
  searchTerm,
  setSearchTerm,
  filterStatus,
  setFilterStatus,
}: any) => {
  const formatBytes = (bytes: any) => {
    const num = parseFloat(bytes)
    if (isNaN(num) || num <= 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(num) / Math.log(k))
    return parseFloat((num / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const filteredTasks = tasks.filter((task: Task) => {
    const matchesSearch =
      task.file_name?.toLowerCase().includes(searchTerm) ||
      task.id?.toLowerCase().includes(searchTerm) ||
      task.url?.toLowerCase().includes(searchTerm)
    const matchesStatus = filterStatus === 'all' || task.status === filterStatus
    return matchesSearch && matchesStatus
  })

  return (
    <div className="space-y-10 animate-in slide-in-from-right-10 duration-700">
      <div className="flex flex-col md:flex-row gap-6 justify-between items-center">
        <div className="relative w-full md:w-96 group">
          <input
            type="text"
            placeholder="Search tasks..."
            className="w-full bg-white dark:bg-zinc-900/60 border border-slate-200 dark:border-white/5 p-4 pl-12 rounded-2xl outline-none focus:ring-4 ring-primary/10 transition-all"
            onChange={(e) => {
              const term = e.target.value.toLowerCase()
              setSearchTerm(term)
            }}
          />
          <div className="absolute left-4 top-1/2 -translate-y-1/2 text-slate-400 group-focus-within:text-primary transition-colors">
            <Activity size={20} />
          </div>
        </div>
        <div className="flex items-center space-x-3 overflow-x-auto pb-2 md:pb-0 scrollbar-hide">
          {['all', 'downloading', 'uploading', 'queued', 'completed', 'failed'].map((s) => (
            <button
              key={s}
              onClick={() => setFilterStatus(s)}
              className={`px-6 py-3 rounded-2xl text-[10px] font-black uppercase tracking-widest transition-all ${filterStatus === s
                ? 'bg-primary text-white shadow-lg shadow-primary/20 scale-105'
                : 'bg-white dark:bg-zinc-900/60 text-slate-400 border border-slate-200 dark:border-white/5 hover:bg-slate-50'
                }`}
            >
              {s}
            </button>
          ))}
        </div>
      </div>

      {filteredTasks.length > 0 ? (
        <div className="grid grid-cols-1 gap-6">
          {filteredTasks.map((task: Task) => (
            <TaskRow
              key={task.id}
              task={task}
              formatBytes={formatBytes}
              onCancel={cancelTask}
            />
          ))}
        </div>
      ) : (
        <div className="glass-card py-32 rounded-[3.5rem] flex flex-col items-center justify-center text-center opacity-40">
          <div className="p-8 bg-slate-100 dark:bg-white/5 rounded-full mb-6">
            <Activity size={48} className="text-slate-400" />
          </div>
          <h4 className="text-xl font-black uppercase tracking-widest text-slate-400">
            No matches found
          </h4>
        </div>
      )}
    </div>
  )
}

const App = () => {
  const [apiToken, setApiToken] = useState<string>(localStorage.getItem('api_token') || '')
  const [loginError, setLoginError] = useState<boolean>(!localStorage.getItem('api_token'))
  const [isDarkMode, setIsDarkMode] = useState<boolean>(false)
  const [loading, setLoading] = useState<boolean>(false)
  const [searchTerm, setSearchTerm] = useState('')
  const [filterStatus, setFilterStatus] = useState('all')

  const { tasks, fetchTasks, cancelTask, setTasks } = useTasks(apiToken)
  const { stats, system, fetchStats, setSystem } = useSystemStats(apiToken)
  const { showConfirm } = usePopups()
  const [settings, setSettings] = useState<any>(null)

  useEffect(() => {
    const interceptor = axios.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response && error.response.status === 401) {
          setLoginError(true)
          setApiToken('')
          localStorage.removeItem('api_token')
        }
        return Promise.reject(error)
      },
    )

    return () => {
      axios.interceptors.response.eject(interceptor)
    }
  }, [])

  useEffect(() => {
    const savedTheme = localStorage.getItem('theme') || 'light'
    const isDark = savedTheme === 'dark'
    setIsDarkMode(isDark)
    if (isDark) document.documentElement.classList.add('dark')
    else document.documentElement.classList.remove('dark')
  }, [])

  const toggleTheme = () => {
    const newMode = !isDarkMode
    setIsDarkMode(newMode)
    localStorage.setItem('theme', newMode ? 'dark' : 'light')
    if (newMode) document.documentElement.classList.add('dark')
    else document.documentElement.classList.remove('dark')
  }

  useEffect(() => {
    if (apiToken && !loginError) {
      setLoading(true)
      const loadData = async () => {
        try {
          await Promise.all([fetchTasks(), fetchStats()])

          try {
            const res = await axios.get('/api/settings', { headers: { 'X-API-Key': apiToken } })
            setSettings(res.data)
          } catch (e) {
            console.error('Settings load error', e)
          }
        } catch (e) {
          console.error('Initial load failed', e)
        } finally {
          setLoading(false)
        }
      }
      loadData()

      let ws: WebSocket | null = null
      let reconnectAttempts = 0
      let reconnectTimeout: number | ReturnType<typeof setTimeout> | null = null

      const connectWs = () => {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        const wsUrl = `${protocol}//${window.location.host}/api/ws?token=${apiToken}`
        ws = new WebSocket(wsUrl)

        ws.onopen = () => {
          console.log('🟢 WS Connected')
          reconnectAttempts = 0
        }

        ws.onmessage = (event) => {
          try {
            const msg = JSON.parse(event.data)
            if (msg.type === 'update') {
              if (msg.data.tasks) setTasks(msg.data.tasks)
              if (msg.data.system) setSystem((prev: any) => ({ ...prev, ...msg.data.system }))
            }
          } catch (e) {
            console.error('WS Parse Error', e)
          }
        }

        ws.onclose = () => {
          reconnectAttempts++
          const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000)
          console.log(`🔴 WS Closed, reconnecting in ${delay / 1000}s... (Attempt ${reconnectAttempts})`)

          if (reconnectTimeout) clearTimeout(reconnectTimeout)
          reconnectTimeout = setTimeout(() => {
            if (apiToken && !loginError) connectWs()
          }, delay)
        }
      }
      connectWs()

      return () => {
        if (reconnectTimeout) clearTimeout(reconnectTimeout)
        if (ws) ws.close()
      }
    }
  }, [apiToken, loginError, fetchTasks, fetchStats, setTasks, setSystem])

  const onLogout = async () => {
    if (
      await showConfirm(
        'Secure Disconnect',
        'Are you sure you want to disconnect from the Cloud Engine? Active sessions will be terminated.',
      )
    ) {
      localStorage.clear()
      window.location.reload()
    }
  }

  const router = createBrowserRouter([
    {
      path: '/login',
      element: !loginError ? <Navigate to="/" /> : <Login setApiToken={setApiToken} setLoginError={setLoginError} />,
    },
    {
      path: '/torrent-select/:id',
      element: <TorrentSelect token={apiToken} />,
    },
    {
      path: '/',
      element: loginError ? <Navigate to="/login" /> : (
        <DashboardLayout
          tasksCount={tasks ? tasks.length : 0}
          isDarkMode={isDarkMode}
          toggleTheme={toggleTheme}
          loading={loading}
          onLogout={onLogout}
        />
      ),
      children: [
        {
          index: true,
          element: <Navigate to="/overview" replace />,
        },
        {
          path: 'overview',
          element: (
            <Overview
              tasks={tasks}
              stats={stats}
              system={system}
              onCancelTask={cancelTask}
              setActiveTab={() => { }}
            />
          ),
        },
        {
          path: 'tasks/history',
          element: <TaskHistory token={apiToken} />,
        },
        {
          path: 'tasks',
          element: (
            <TasksPage
              tasks={tasks}
              cancelTask={cancelTask}
              searchTerm={searchTerm}
              setSearchTerm={setSearchTerm}
              filterStatus={filterStatus}
              setFilterStatus={setFilterStatus}
            />
          ),
        },
        {
          path: 'files/*',
          element: <Explorer token={apiToken} />,
        },
        {
          path: 'files',
          element: <Explorer token={apiToken} />,
        },
        {
          path: 'analytics',
          element: <Analytics token={apiToken} isDarkMode={isDarkMode} />,
        },
        {
          path: 'logs',
          element: <Logs token={apiToken} />,
        },
        {
          path: 'users',
          element: <Users apiToken={apiToken} />,
        },
        {
          path: 'settings',
          element: <Settings token={apiToken} initialSettings={settings} />,
        },
      ],
    },
  ])

  return <RouterProvider router={router} />
}

export default App

