// Design System Constants
// Resource-to-Code Correlation Engine Dashboard

// Color Palette
export const colors = {
  // Primary Brand Colors
  primary: {
    50: '#eff6ff',
    100: '#dbeafe',
    200: '#bfdbfe',
    300: '#93c5fd',
    400: '#60a5fa',
    500: '#3b82f6',
    600: '#2563eb',
    700: '#1d4ed8',
    800: '#1e40af',
    900: '#1e3a8a',
  },
  
  // Accent Colors
  accent: {
    // Purple for RAM/Memory
    purple: {
      50: '#f5f3ff',
      100: '#ede9fe',
      200: '#ddd6fe',
      300: '#c4b5fd',
      400: '#a78bfa',
      500: '#8b5cf6',
      600: '#7c3aed',
      700: '#6d28d9',
      800: '#5b21b6',
      900: '#4c1d95',
    },
    // Amber for warnings
    amber: {
      50: '#fffbeb',
      100: '#fef3c7',
      200: '#fde68a',
      300: '#fcd34d',
      400: '#fbbf24',
      500: '#f59e0b',
      600: '#d97706',
      700: '#b45309',
      800: '#92400e',
      900: '#78350f',
    },
    // Red for errors/critical
    red: {
      50: '#fef2f2',
      100: '#fee2e2',
      200: '#fecaca',
      300: '#fca5a5',
      400: '#f87171',
      500: '#ef4444',
      600: '#dc2626',
      700: '#b91c1c',
      800: '#991b1b',
      900: '#7f1d1d',
    },
    // Green for success
    green: {
      50: '#f0fdf4',
      100: '#dcfce7',
      200: '#bbf7d0',
      300: '#86efac',
      400: '#4ade80',
      500: '#22c55e',
      600: '#16a34a',
      700: '#15803d',
      800: '#166534',
      900: '#14532d',
    },
    // Cyan for info
    cyan: {
      50: '#ecfeff',
      100: '#cffafe',
      200: '#a5f3fc',
      300: '#67e8f9',
      400: '#22d3ee',
      500: '#06b6d4',
      600: '#0891b2',
      700: '#0e7490',
      800: '#155e75',
      900: '#164e63',
    },
  },
  
  // Neutral Grays
  gray: {
    50: '#f9fafb',
    100: '#f3f4f6',
    200: '#e5e7eb',
    300: '#d1d5db',
    400: '#9ca3af',
    500: '#6b7280',
    600: '#4b5563',
    700: '#374151',
    800: '#1f2937',
    900: '#111827',
    950: '#030712',
  },
  
  // Chart Colors
  chart: {
    cpu: '#2563eb',
    ram: '#8b5cf6',
    cpuArea: 'rgba(37, 99, 235, 0.2)',
    ramArea: 'rgba(139, 92, 246, 0.2)',
    grid: '#e5e7eb',
    tooltip: '#ffffff',
  },
  
  // Status Colors
  status: {
    critical: '#ef4444',
    warning: '#f59e0b',
    success: '#22c55e',
    info: '#3b82f6',
  },
};

// Typography Scale
export const typography = {
  // Font Families
  fontFamily: {
    sans: ['Inter', 'system-ui', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'sans-serif'],
    mono: ['JetBrains Mono', 'Fira Code', 'Consolas', 'Monaco', 'monospace'],
  },
  
  // Font Sizes (in rem)
  fontSize: {
    xs: '0.75rem',    // 12px
    sm: '0.875rem',   // 14px
    base: '1rem',     // 16px
    lg: '1.125rem',   // 18px
    xl: '1.25rem',    // 20px
    '2xl': '1.5rem', // 24px
    '3xl': '1.875rem', // 30px
    '4xl': '2.25rem', // 36px
  },
  
  // Font Weights
  fontWeight: {
    normal: '400',
    medium: '500',
    semibold: '600',
    bold: '700',
  },
  
  // Line Heights
  lineHeight: {
    tight: '1.25',
    normal: '1.5',
    relaxed: '1.75',
  },
};

// Spacing System (based on 4px base unit)
export const spacing = {
  0: '0',
  1: '0.25rem',   // 4px
  2: '0.5rem',    // 8px
  3: '0.75rem',   // 12px
  4: '1rem',     // 16px
  5: '1.25rem',  // 20px
  6: '1.5rem',    // 24px
  8: '2rem',      // 32px
  10: '2.5rem',  // 40px
  12: '3rem',    // 48px
  16: '4rem',    // 64px
};

// Border Radius
export const borderRadius = {
  none: '0',
  sm: '0.25rem',    // 4px
  DEFAULT: '0.375rem', // 6px
  md: '0.5rem',     // 8px
  lg: '0.75rem',    // 12px
  xl: '1rem',       // 16px
  '2xl': '1.5rem', // 24px
  full: '9999px',
};

// Shadows
export const shadows = {
  sm: '0 1px 2px 0 rgba(0, 0, 0, 0.05)',
  DEFAULT: '0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px -1px rgba(0, 0, 0, 0.1)',
  md: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -2px rgba(0, 0, 0, 0.1)',
  lg: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -4px rgba(0, 0, 0, 0.1)',
  xl: '0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 8px 10px -6px rgba(0, 0, 0, 0.1)',
};

// Transitions
export const transitions = {
  fast: '150ms ease-in-out',
  normal: '200ms ease-in-out',
  slow: '300ms ease-in-out',
};

// Breakpoints (must match Tailwind config)
export const breakpoints = {
  sm: '640px',
  md: '768px',
  lg: '1024px',
  xl: '1280px',
  '2xl': '1536px',
};

// Z-Index Scale
export const zIndex = {
  base: '0',
  dropdown: '1000',
  sticky: '1100',
  fixed: '1200',
  modalBackdrop: '1300',
  modal: '1400',
  popover: '1500',
  tooltip: '1600',
};

// Component-specific exports
export const components = {
  // Card styles
  card: {
    padding: '1.5rem',
    borderRadius: '0.75rem',
    backgroundColor: '#ffffff',
    shadow: shadows.md,
    border: `1px solid ${colors.gray[200]}`,
  },
  
  // Button variants
  button: {
    primary: {
      backgroundColor: colors.primary[600],
      color: '#ffffff',
      hoverBackgroundColor: colors.primary[700],
      activeBackgroundColor: colors.primary[800],
    },
    secondary: {
      backgroundColor: colors.gray[100],
      color: colors.gray[700],
      hoverBackgroundColor: colors.gray[200],
      activeBackgroundColor: colors.gray[300],
    },
    outline: {
      backgroundColor: 'transparent',
      color: colors.primary[600],
      borderColor: colors.primary[600],
      hoverBackgroundColor: colors.primary[50],
    },
  },
  
  // Input styles
  input: {
    padding: '0.5rem 0.75rem',
    borderRadius: '0.375rem',
    borderColor: colors.gray[300],
    focusRingColor: colors.primary[500],
    placeholderColor: colors.gray[400],
  },
  
  // Badge variants
  badge: {
    critical: {
      backgroundColor: colors.accent.red[100],
      color: colors.accent.red[800],
    },
    warning: {
      backgroundColor: colors.accent.amber[100],
      color: colors.accent.amber[800],
    },
    success: {
      backgroundColor: colors.accent.green[100],
      color: colors.accent.green[800],
    },
    info: {
      backgroundColor: colors.accent.cyan[100],
      color: colors.accent.cyan[800],
    },
  },
};

// Animation Keyframes
export const animations = {
  pulse: {
    '0%, 100%': { opacity: '1' },
    '50%': { opacity: '0.5' },
  },
  shimmer: {
    '0%': { backgroundPosition: '-200% 0' },
    '100%': { backgroundPosition: '200% 0' },
  },
};

// Dark mode colors (for future support)
export const darkModeColors = {
  background: {
    primary: '#0f172a',
    secondary: '#1e293b',
    tertiary: '#334155',
  },
  text: {
    primary: '#f8fafc',
    secondary: '#cbd5e1',
    muted: '#94a3b8',
  },
  border: '#334155',
};

export default {
  colors,
  typography,
  spacing,
  borderRadius,
  shadows,
  transitions,
  breakpoints,
  zIndex,
  components,
  animations,
  darkModeColors,
};