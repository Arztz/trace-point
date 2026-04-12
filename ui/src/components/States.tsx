import { FiInbox, FiAlertCircle, FiRefreshCw, FiHome } from 'react-icons/fi';
import { Link } from 'react-router-dom';

interface EmptyStateProps {
  icon?: 'inbox' | 'alert' | 'search' | 'chart';
  title: string;
  description: string;
  action?: {
    label: string;
    onClick?: () => void;
    href?: string;
  };
}

const icons = {
  inbox: FiInbox,
  alert: FiAlertCircle,
  search: FiInbox,
  chart: FiInbox,
};

export function EmptyState({ icon = 'inbox', title, description, action }: EmptyStateProps) {
  const Icon = icons[icon];
  
  return (
    <div className="flex flex-col items-center justify-center py-16 px-4 text-center">
      <div className="w-20 h-20 bg-gray-100 rounded-full flex items-center justify-center mb-6">
        <Icon className="w-10 h-10 text-gray-400" />
      </div>
      <h3 className="text-lg font-semibold text-gray-900 mb-2">{title}</h3>
      <p className="text-gray-500 max-w-sm mb-6">{description}</p>
      {action && (
        action.href ? (
          <Link to={action.href} className="btn btn-primary">
            {action.label}
          </Link>
        ) : (
          <button onClick={action.onClick} className="btn btn-primary">
            {action.label}
          </button>
        )
      )}
    </div>
  );
}

interface ErrorStateProps {
  title?: string;
  message: string;
  onRetry?: () => void;
}

export function ErrorState({ 
  title = 'Something went wrong', 
  message, 
  onRetry 
}: ErrorStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 px-4 text-center">
      <div className="w-20 h-20 bg-red-50 rounded-full flex items-center justify-center mb-6">
        <FiAlertCircle className="w-10 h-10 text-red-500" />
      </div>
      <h3 className="text-lg font-semibold text-gray-900 mb-2">{title}</h3>
      <p className="text-gray-500 max-w-sm mb-6">{message}</p>
      {onRetry && (
        <button onClick={onRetry} className="btn btn-outline">
          <FiRefreshCw className="w-4 h-4 mr-2" />
          Try again
        </button>
      )}
    </div>
  );
}

interface LoadingStateProps {
  text?: string;
}

export function LoadingState({ text = 'Loading...' }: LoadingStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 px-4">
      <div className="w-10 h-10 border-4 border-primary-100 border-t-primary-600 rounded-full animate-spin mb-4" />
      <p className="text-gray-500 text-sm">{text}</p>
    </div>
  );
}

export function PageNotFound() {
  return (
    <div className="flex flex-col items-center justify-center min-h-[60vh] px-4 text-center">
      <div className="w-24 h-24 bg-gray-100 rounded-full flex items-center justify-center mb-6">
        <FiHome className="w-12 h-12 text-gray-400" />
      </div>
      <h1 className="text-4xl font-bold text-gray-900 mb-3">404</h1>
      <h2 className="text-xl font-semibold text-gray-700 mb-2">Page Not Found</h2>
      <p className="text-gray-500 max-w-md mb-8">
        The page you're looking for doesn't exist or has been moved.
      </p>
      <Link to="/" className="btn btn-primary">
        Go to Dashboard
      </Link>
    </div>
  );
}

export default {
  EmptyState,
  ErrorState,
  LoadingState,
  PageNotFound,
};