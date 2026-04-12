import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  AreaChart,
  Area,
  Legend,
} from 'recharts';
import type { TimelineData } from '../types';

interface TimelineChartProps {
  data: TimelineData;
}

interface CustomTooltipProps {
  active?: boolean;
  payload?: Array<{
    name: string;
    value: number;
    color: string;
    dataKey: string;
  }>;
  label?: string;
}

function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleTimeString('en-US', { 
    hour: '2-digit', 
    minute: '2-digit',
    hour12: false 
  });
}

function formatDate(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
  });
}

const CustomTooltip = ({ active, payload, label }: CustomTooltipProps) => {
  if (active && payload && payload.length) {
    return (
      <div className="bg-white border border-gray-200 rounded-lg shadow-lg p-3 min-w-[160px]">
        <p className="text-xs font-medium text-gray-500 mb-2">{label}</p>
        {payload.map((entry) => (
          <div key={entry.dataKey} className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-1.5">
              <span 
                className="w-2 h-2 rounded-full" 
                style={{ backgroundColor: entry.color }}
              />
              <span className="text-xs text-gray-600">
                {entry.dataKey === 'cpu' ? 'CPU' : 'RAM'}
              </span>
            </div>
            <span className="text-sm font-semibold text-gray-900">
              {entry.dataKey === 'cpu' 
                ? `${entry.value.toFixed(1)}%` 
                : `${entry.value.toFixed(0)} MB`}
            </span>
          </div>
        ))}
      </div>
    );
  }
  return null;
};

export default function TimelineChart({ data }: TimelineChartProps) {
  const chartData = data.dataPoints.map((point) => ({
    time: formatTimestamp(point.timestamp),
    date: formatDate(point.timestamp),
    cpu: point.cpu,
    ram: point.ram,
  }));

  const avgCpu = chartData.length > 0
    ? (chartData.reduce((sum, d) => sum + d.cpu, 0) / chartData.length).toFixed(1)
    : '0';
  
  const avgRam = chartData.length > 0
    ? (chartData.reduce((sum, d) => sum + d.ram, 0) / chartData.length).toFixed(0)
    : '0';

  const maxCpu = chartData.length > 0
    ? Math.max(...chartData.map(d => d.cpu)).toFixed(1)
    : '0';
  
  const maxRam = chartData.length > 0
    ? Math.max(...chartData.map(d => d.ram)).toFixed(0)
    : '0';

  return (
    <div className="space-y-4">
      {/* Stats Summary */}
      <div className="grid grid-cols-4 gap-4">
        <div className="bg-primary-50 rounded-lg p-3">
          <p className="text-xs text-primary-600 font-medium">Avg CPU</p>
          <p className="text-xl font-bold text-primary-700">{avgCpu}%</p>
        </div>
        <div className="bg-purple-50 rounded-lg p-3">
          <p className="text-xs text-purple-600 font-medium">Avg RAM</p>
          <p className="text-xl font-bold text-purple-700">{avgRam} MB</p>
        </div>
        <div className="bg-red-50 rounded-lg p-3">
          <p className="text-xs text-red-600 font-medium">Max CPU</p>
          <p className="text-xl font-bold text-red-700">{maxCpu}%</p>
        </div>
        <div className="bg-amber-50 rounded-lg p-3">
          <p className="text-xs text-amber-600 font-medium">Max RAM</p>
          <p className="text-xl font-bold text-amber-700">{maxRam} MB</p>
        </div>
      </div>

      {/* Chart */}
      <div className="h-72">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart 
            data={chartData} 
            margin={{ top: 10, right: 30, left: 0, bottom: 0 }}
          >
            <defs>
              <linearGradient id="cpuGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#2563eb" stopOpacity={0.3}/>
                <stop offset="95%" stopColor="#2563eb" stopOpacity={0}/>
              </linearGradient>
              <linearGradient id="ramGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#8b5cf6" stopOpacity={0.3}/>
                <stop offset="95%" stopColor="#8b5cf6" stopOpacity={0}/>
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" vertical={false} />
            <XAxis
              dataKey="time"
              tick={{ fontSize: 11, fill: '#6b7280' }}
              tickLine={{ stroke: '#e5e7eb' }}
              axisLine={{ stroke: '#e5e7eb' }}
              interval="preserveStartEnd"
              minTickGap={60}
            />
            <YAxis
              yAxisId="cpu"
              tick={{ fontSize: 11, fill: '#6b7280' }}
              tickLine={{ stroke: '#e5e7eb' }}
              axisLine={{ stroke: '#e5e7eb' }}
              tickFormatter={(value) => `${value}%`}
              domain={[0, 'auto']}
            />
            <YAxis
              yAxisId="ram"
              orientation="right"
              tick={{ fontSize: 11, fill: '#6b7280' }}
              tickLine={{ stroke: '#e5e7eb' }}
              axisLine={{ stroke: '#e5e7eb' }}
              tickFormatter={(value) => `${value}`}
              domain={[0, 'auto']}
            />
            <Tooltip content={<CustomTooltip />} />
            <Legend 
              verticalAlign="top" 
              height={36}
              iconType="circle"
              iconSize={8}
              formatter={(value) => (
                <span className="text-sm font-medium text-gray-700">
                  {value === 'cpu' ? 'CPU (%)' : 'RAM (MB)'}
                </span>
              )}
            />
            <Area
              yAxisId="cpu"
              type="monotone"
              dataKey="cpu"
              stroke="#2563eb"
              strokeWidth={2}
              fill="url(#cpuGradient)"
              dot={false}
              activeDot={{ r: 5, stroke: '#2563eb', strokeWidth: 2, fill: '#fff' }}
              name="cpu"
            />
            <Area
              yAxisId="ram"
              type="monotone"
              dataKey="ram"
              stroke="#8b5cf6"
              strokeWidth={2}
              fill="url(#ramGradient)"
              dot={false}
              activeDot={{ r: 5, stroke: '#8b5cf6', strokeWidth: 2, fill: '#fff' }}
              name="ram"
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}