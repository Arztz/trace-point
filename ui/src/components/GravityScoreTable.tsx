import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { FiServer, FiCpu, FiDatabase, FiTrendingUp, FiAlertCircle, FiChevronDown, FiChevronUp, FiArrowUp, FiArrowDown } from 'react-icons/fi';
import { api } from '../services/api';
import type { GravityScore } from '../types/gravity';
import { ErrorState, LoadingState } from './States';

interface GravityScoreTableProps {
  namespaceFilter?: string;
  minScoreFilter?: number;
}

type SortField = 'serviceName' | 'cpuScore' | 'memoryScore' | 'totalScore';
type SortDirection = 'asc' | 'desc';

interface TableState {
  sortField: SortField;
  sortDirection: SortDirection;
}

function getScoreClass(score: number): string {
  if (score >= 80) return 'text-red-600 bg-red-50';
  if (score >= 60) return 'text-amber-600 bg-amber-50';
  if (score >= 40) return 'text-yellow-600 bg-yellow-50';
  return 'text-green-600 bg-green-50';
}

function getScoreLabel(score: number): string {
  if (score >= 80) return 'Critical';
  if (score >= 60) return 'High';
  if (score >= 40) return 'Medium';
  return 'Low';
}

function formatScoreValue(value: number): string {
  return value.toFixed(1);
}

export default function GravityScoreTable({ namespaceFilter, minScoreFilter }: GravityScoreTableProps) {
  const [sort, setSort] = useState<TableState>({ sortField: 'totalScore', sortDirection: 'desc' });

  const { data, isLoading, error } = useQuery<GravityScore[]>({
    queryKey: ['gravityScores', namespaceFilter, minScoreFilter],
    queryFn: () => api.gravityScores.list({ 
      namespace: namespaceFilter, 
      minScore: minScoreFilter 
    }).then(res => res.scores),
  });

  const handleSort = (field: SortField) => {
    setSort(prev => ({
      sortField: field,
      sortDirection: prev.sortField === field && prev.sortDirection === 'desc' ? 'asc' : 'desc',
    }));
  };

  const sortedData = data ? [...data].sort((a, b) => {
    const aVal = Number(a[sort.sortField]);
    const bVal = Number(b[sort.sortField]);
    const modifier = sort.sortDirection === 'asc' ? 1 : -1;
    return (aVal - bVal) * modifier;
  }) : [];

  const SortIcon = ({ field }: { field: SortField }) => {
    if (sort.sortField !== field) return null;
    return sort.sortDirection === 'asc' 
      ? <FiArrowUp className="w-3 h-3 ml-1" />
      : <FiArrowDown className="w-3 h-3 ml-1" />;
  };

  if (isLoading) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <LoadingState text="Loading resource scores..." />
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <ErrorState 
          title="Failed to load gravity scores" 
          message="Could not retrieve resource gravity scores."
        />
      </div>
    );
  }

  if (!sortedData || sortedData.length === 0) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <div className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mb-4">
            <FiServer className="w-8 h-8 text-gray-400" />
          </div>
          <p className="text-gray-500 font-medium">No gravity scores available</p>
          <p className="text-gray-400 text-sm mt-1">Resource scores will appear here once analyzed</p>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
      {/* Table Header */}
      <div className="bg-gray-50 px-6 py-4 border-b border-gray-200">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-primary-100 rounded-lg">
            <FiTrendingUp className="w-5 h-5 text-primary-600" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-gray-900">Resource Gravity Scores</h2>
            <p className="text-sm text-gray-500">Service resource consumption ranking</p>
          </div>
        </div>
      </div>

      {/* Table */}
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="bg-gray-50 border-b border-gray-200">
            <tr>
              <th 
                className="px-6 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                onClick={() => handleSort('serviceName')}
              >
                <div className="flex items-center gap-1">
                  Service
                  <SortIcon field="serviceName" />
                </div>
              </th>
              <th className="px-6 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">
                Namespace
              </th>
              <th 
                className="px-6 py-3 text-right text-xs font-semibold text-gray-600 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                onClick={() => handleSort('cpuScore')}
              >
                <div className="flex items-center justify-end gap-1">
                  CPU Score
                  <SortIcon field="cpuScore" />
                </div>
              </th>
              <th 
                className="px-6 py-3 text-right text-xs font-semibold text-gray-600 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                onClick={() => handleSort('memoryScore')}
              >
                <div className="flex items-center justify-end gap-1">
                  Memory Score
                  <SortIcon field="memoryScore" />
                </div>
              </th>
              <th 
                className="px-6 py-3 text-right text-xs font-semibold text-gray-600 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                onClick={() => handleSort('totalScore')}
              >
                <div className="flex items-center justify-end gap-1">
                  Total Score
                  <SortIcon field="totalScore" />
                </div>
              </th>
              <th className="px-6 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">
                Suggestion
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {sortedData.map((score, idx) => (
              <tr 
                key={idx} 
                className="hover:bg-gray-50 transition-colors"
              >
                <td className="px-6 py-4 whitespace-nowrap">
                  <div className="flex items-center gap-3">
                    <div className="p-2 bg-gray-100 rounded-lg">
                      <FiServer className="w-4 h-4 text-gray-600" />
                    </div>
                    <span className="text-sm font-medium text-gray-900 font-mono">
                      {score.serviceName}
                    </span>
                  </div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <span className="text-sm text-gray-600 font-mono">
                    {score.namespace}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right">
                  <span className={`inline-flex items-center px-2.5 py-1 rounded-full text-xs font-semibold ${getScoreClass(score.cpuScore)}`}>
                    {formatScoreValue(score.cpuScore)}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right">
                  <span className={`inline-flex items-center px-2.5 py-1 rounded-full text-xs font-semibold ${getScoreClass(score.memoryScore)}`}>
                    {formatScoreValue(score.memoryScore)}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right">
                  <span className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-semibold ${getScoreClass(score.totalScore)}`}>
                    {formatScoreValue(score.totalScore)}
                  </span>
                </td>
                <td className="px-6 py-4">
                  <div className="flex items-start gap-2 max-w-xs">
                    <FiAlertCircle className="w-4 h-4 text-amber-500 mt-0.5 flex-shrink-0" />
                    <span className="text-sm text-gray-600">
                      {score.suggestion}
                    </span>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}