import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { FiActivity, FiTrendingUp, FiSettings, FiHelpCircle, FiGithub } from 'react-icons/fi';
import Dashboard from './pages/Dashboard';

function App() {
  return (
    <BrowserRouter>
      <div className="min-h-screen bg-gray-50">
        {/* Header */}
        <header className="bg-gradient-to-r from-gray-900 to-gray-800 shadow-lg">
          <div className="max-w-full mx-4 sm:mx-6 lg:mx-8">
            <div className="flex items-center justify-between h-16">
              {/* Logo & Title */}
              <div className="flex items-center gap-3">
                <div className="p-2 bg-white/10 rounded-lg backdrop-blur-sm">
                  <FiActivity className="w-6 h-6 text-primary-400" />
                </div>
                <div>
                  <h1 className="text-xl font-bold text-white tracking-tight">
                    Trace Point
                  </h1>
                  <p className="text-xs text-gray-400 hidden sm:block">
                    Resource to Code Correlation Engine
                  </p>
                </div>
              </div>

              {/* Navigation */}
              <nav className="hidden md:flex items-center gap-1">
                <a 
                  href="#" 
                  className="px-3 py-2 text-sm font-medium text-gray-300 hover:text-white hover:bg-white/10 rounded-md transition-colors"
                >
                  Dashboard
                </a>
                <a 
                  href="#" 
                  className="px-3 py-2 text-sm font-medium text-gray-400 hover:text-white hover:bg-white/10 rounded-md transition-colors"
                >
                  Spikes
                </a>
                <a 
                  href="#" 
                  className="px-3 py-2 text-sm font-medium text-gray-400 hover:text-white hover:bg-white/10 rounded-md transition-colors"
                >
                  Services
                </a>
              </nav>

              {/* Right Actions */}
              <div className="flex items-center gap-2">
                <button 
                  className="p-2 text-gray-400 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                  title="Settings"
                >
                  <FiSettings className="w-5 h-5" />
                </button>
                <button 
                  className="p-2 text-gray-400 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                  title="Help"
                >
                  <FiHelpCircle className="w-5 h-5" />
                </button>
                <div className="ml-2 pl-4 border-l border-gray-700">
                  <a 
                    href="#" 
                    className="p-2 text-gray-400 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                    title="GitHub"
                  >
                    <FiGithub className="w-5 h-5" />
                  </a>
                </div>
              </div>
            </div>
          </div>
        </header>

        {/* Breadcrumb */}
        <div className="bg-white border-b border-gray-200">
          <div className="max-w-full mx-4 sm:mx-6 lg:mx-8">
            <div className="flex items-center gap-2 h-10 text-sm">
              <span className="text-gray-500">Home</span>
              <span className="text-gray-400">/</span>
              <span className="text-gray-900 font-medium">Dashboard</span>
            </div>
          </div>
        </div>

        {/* Main Content */}
        <main className="flex-1 flex flex-col max-w-full mx-4 sm:mx-6 lg:mx-8 py-8">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </main>

        {/* Footer */}
        <footer className="bg-white border-t border-gray-200 mt-auto">
          <div className="max-w-full mx-4 sm:mx-6 lg:mx-8 py-6">
            <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
              <div className="flex items-center gap-2 text-sm text-gray-500">
                <FiActivity className="w-4 h-4 text-primary-500" />
                <span>Trace Point v1.0.0</span>
              </div>
              <div className="flex items-center gap-4 text-sm text-gray-400">
                <a href="#" className="hover:text-gray-600 transition-colors">Documentation</a>
                <a href="#" className="hover:text-gray-600 transition-colors">API</a>
                <a href="#" className="hover:text-gray-600 transition-colors">Status</a>
              </div>
            </div>
          </div>
        </footer>
      </div>
    </BrowserRouter>
  );
}

export default App;