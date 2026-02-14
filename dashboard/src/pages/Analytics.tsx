import { useEffect } from 'react'
import { TrendingUp, BarChart3 } from 'lucide-react'
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
  Cell,
} from 'recharts'
import AnalyticsStat from '../components/Stats/AnalyticsStat'
import useAnalytics from '../hooks/useAnalytics'

const CustomTooltip = ({ active, payload, label }: any) => {
  if (active && payload && payload.length) {
    return (
      <div className="bg-white/90 dark:bg-zinc-900/90 backdrop-blur-xl p-4 border border-slate-200 dark:border-white/5 rounded-2xl shadow-2xl shadow-black/20">
        <p className="text-[10px] font-black text-slate-500 dark:text-slate-400 uppercase tracking-widest mb-1">
          {label}
        </p>
        <p className="text-lg font-black text-primary">
          {payload[0].value} <span className="text-[10px] uppercase">Tasks</span>
        </p>
      </div>
    )
  }
  return null
}

interface AnalyticsProps {
  token: string
  isDarkMode: boolean
}

const Analytics: React.FC<AnalyticsProps> = ({ token, isDarkMode }) => {
  const { analyticsData, fetchAnalytics } = useAnalytics(token)

  useEffect(() => {
    fetchAnalytics()
  }, [fetchAnalytics])

  return (
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
                  <CartesianGrid
                    strokeDasharray="3 3"
                    vertical={false}
                    stroke={isDarkMode ? '#ffffff05' : '#00000005'}
                  />
                  <XAxis
                    dataKey="Date"
                    axisLine={false}
                    tickLine={false}
                    tick={{
                      fill: isDarkMode ? '#94a3b8' : '#64748b',
                      fontSize: 10,
                      fontWeight: 900,
                    }}
                    tickFormatter={(str) =>
                      new Date(str).toLocaleDateString('en-US', { weekday: 'short' })
                    }
                  />
                  <YAxis hide />
                  <Tooltip
                    content={<CustomTooltip />}
                    cursor={{ stroke: '#3b82f6', strokeWidth: 2 }}
                  />
                  <Area
                    type="monotone"
                    dataKey="TotalTasks"
                    stroke="#3b82f6"
                    strokeWidth={4}
                    fillOpacity={1}
                    fill="url(#colorTasks)"
                  />
                </AreaChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex items-center justify-center h-full text-slate-400">
                No Data Available
              </div>
            )}
          </div>
        </div>

        <div className="glass-card p-12 rounded-[4rem] shadow-2xl space-y-8">
          <div className="flex items-center justify-between px-2">
            <h3 className="text-3xl font-black text-slate-900 dark:text-white">
              Distribution Ratio
            </h3>
            <BarChart3 className="text-indigo-400" size={28} />
          </div>
          <div className="h-[350px] w-full mt-8" style={{ minHeight: '350px' }}>
            {analyticsData.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%" minHeight={350}>
                <BarChart data={analyticsData}>
                  <CartesianGrid
                    strokeDasharray="3 3"
                    vertical={false}
                    stroke={isDarkMode ? '#ffffff05' : '#00000005'}
                  />
                  <XAxis
                    dataKey="Date"
                    axisLine={false}
                    tickLine={false}
                    tick={{
                      fill: isDarkMode ? '#94a3b8' : '#64748b',
                      fontSize: 10,
                      fontWeight: 900,
                    }}
                    tickFormatter={(str) => new Date(str).getDate().toString()}
                  />
                  <Tooltip
                    content={<CustomTooltip />}
                    cursor={{ fill: isDarkMode ? '#ffffff05' : '#00000005' }}
                  />
                  <Bar dataKey="TotalTasks" radius={[10, 10, 10, 10]} barSize={32}>
                    {analyticsData.map((_entry, index) => (
                      <Cell
                        key={`cell-${index}`}
                        fill={index === analyticsData.length - 1 ? '#3b82f6' : '#94a3b830'}
                      />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex items-center justify-center h-full text-slate-400">
                No Data Available
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
        <AnalyticsStat
          label="Peak Performance"
          value={Math.max(...analyticsData.map((d) => d.TotalTasks), 0)}
          unit="Tasks/Day"
        />
        <AnalyticsStat
          label="Weekly Volume"
          value={analyticsData.reduce((acc, curr) => acc + curr.TotalTasks, 0)}
          unit="Aggregate"
        />
        <AnalyticsStat label="Node Efficiency" value="98.4%" unit="Realtime" />
      </div>
    </div>
  )
}

export default Analytics
