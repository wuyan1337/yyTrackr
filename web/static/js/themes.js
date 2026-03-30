// SubTrackr Theme System
const themes = {
    default: {
        name: 'Default',
        description: 'Clean and professional',
        colors: {
            primary: '#3b82f6',
            primaryHover: '#2563eb',
            secondary: '#64748b',
            success: '#10b981',
            warning: '#f59e0b',
            danger: '#ef4444',
            background: '#f9fafb',
            surface: '#ffffff',
            surfaceHover: '#f3f4f6',
            text: '#111827',
            textSecondary: '#6b7280',
            border: '#e5e7eb',
        }
    },
    dark: {
        name: 'Dark',
        description: 'Easy on the eyes',
        colors: {
            primary: '#3b82f6',
            primaryHover: '#60a5fa',
            secondary: '#64748b',
            success: '#10b981',
            warning: '#f59e0b',
            danger: '#ef4444',
            background: '#111827',
            surface: '#1f2937',
            surfaceHover: '#374151',
            text: '#f9fafb',
            textSecondary: '#9ca3af',
            border: '#374151',
        }
    },
    'dark-classic': {
        name: 'Dark Classic',
        description: 'Original dark mode',
        useTailwindDark: true,  // Special flag to use Tailwind's dark mode
        colors: {
            primary: '#3b82f6',
            primaryHover: '#60a5fa',
            secondary: '#64748b',
            success: '#10b981',
            warning: '#f59e0b',
            danger: '#ef4444',
            background: '#111827',
            surface: '#1f2937',
            surfaceHover: '#374151',
            text: '#f9fafb',
            textSecondary: '#9ca3af',
            border: '#374151',
        }
    },
    christmas: {
        name: 'Christmas',
        description: 'Festive and jolly! 🎄',
        colors: {
            primary: '#c41e3a',      // Christmas red
            primaryHover: '#a01729',
            secondary: '#165b33',     // Forest green
            success: '#10b981',
            warning: '#ffd700',       // Gold
            danger: '#ef4444',
            background: '#fef3f3',    // Light snow white with hint of red
            surface: '#ffffff',
            surfaceHover: '#fef2f2',
            text: '#1f2937',
            textSecondary: '#6b7280',
            border: '#fecaca',        // Light red border
        },
        special: {
            accent: '#ffd700',        // Gold accents
            snow: '#ffffff',
            holly: '#165b33',
            berry: '#c41e3a'
        }
    },
    midnight: {
        name: 'Midnight',
        description: 'Deep and mysterious',
        colors: {
            primary: '#8b5cf6',       // Purple
            primaryHover: '#7c3aed',
            secondary: '#64748b',
            success: '#10b981',
            warning: '#f59e0b',
            danger: '#ef4444',
            background: '#0f172a',    // Very dark blue
            surface: '#1e293b',
            surfaceHover: '#334155',
            text: '#f1f5f9',
            textSecondary: '#94a3b8',
            border: '#334155',
        }
    },
    ocean: {
        name: 'Ocean',
        description: 'Cool and refreshing',
        colors: {
            primary: '#0891b2',       // Cyan
            primaryHover: '#06b6d4',
            secondary: '#64748b',
            success: '#10b981',
            warning: '#f59e0b',
            danger: '#ef4444',
            background: '#f0f9ff',    // Light cyan
            surface: '#ffffff',
            surfaceHover: '#e0f2fe',
            text: '#0c4a6e',
            textSecondary: '#475569',
            border: '#bae6fd',
        }
    },
    'gal-violet': {
        name: 'Gal Violet',
        description: 'Anime glassmorphism with sakura neon accents',
        colors: {
            primary: '#f472b6',
            primaryHover: '#ec4899',
            secondary: '#c084fc',
            success: '#34d399',
            warning: '#f59e0b',
            danger: '#fb7185',
            background: '#090b19',
            surface: '#111827',
            surfaceHover: '#1f2937',
            text: '#fdf2f8',
            textSecondary: '#c4b5fd',
            border: '#312e81',
        }
    }
};

// Apply theme to document
function applyTheme(themeName) {
    const theme = themes[themeName] || themes['dark-classic'];
    const root = document.documentElement;

    // Set theme data attribute
    root.setAttribute('data-theme', themeName);

    // Handle Tailwind dark mode for dark-classic theme
    if (theme.useTailwindDark) {
        root.classList.add('dark');
    } else {
        root.classList.remove('dark');
    }

    // Apply CSS variables
    Object.entries(theme.colors).forEach(([key, value]) => {
        root.style.setProperty(`--theme-${key}`, value);
    });

    // Apply special properties for Christmas theme
    if (themeName === 'christmas' && theme.special) {
        Object.entries(theme.special).forEach(([key, value]) => {
            root.style.setProperty(`--theme-special-${key}`, value);
        });

        // Enable snow animation
        enableSnowfall();
    } else {
        // Disable snow animation for other themes
        disableSnowfall();
    }

    // Save theme preference to localStorage AND server
    localStorage.setItem('subtrackr-theme', themeName);
    if (!isAuthPage()) {
        saveThemePreference(themeName);
    }
}

function isAuthPage() {
    const path = window.location.pathname;
    return path === '/login' || path === '/register';
}

// Save theme preference to server
function saveThemePreference(themeName) {
    fetch('/api/settings/theme', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ theme: themeName })
    })
    .then(response => {
        if (!response.ok && response.status !== 401) {
            console.error('Failed to save theme:', response.status);
        }
    })
    .catch(err => console.error('Failed to save theme:', err));
}

// Get theme from localStorage or server
function getStoredTheme() {
    // First check localStorage for instant access
    const localTheme = localStorage.getItem('subtrackr-theme');
    if (localTheme) {
        return Promise.resolve(localTheme);
    }

    // Fall back to server
    return fetch('/api/settings/theme')
        .then(response => response.json())
        .then(data => {
            const theme = data.theme || 'dark-classic';
            localStorage.setItem('subtrackr-theme', theme);
            return theme;
        })
        .catch(err => {
            console.error('Failed to load theme:', err);
            return 'dark-classic';
        });
}

// Load saved theme on page load
function loadSavedTheme() {
    getStoredTheme().then(themeName => {
        applyTheme(themeName);
    });
}

function applyUIPersonalization(config) {
    const root = document.documentElement;
    const body = document.body;
    const enabled = !config || config.enable_chibi_stickers !== false;

    root.classList.toggle('stickers-disabled', !enabled);
    body && body.classList.toggle('stickers-disabled', !enabled);

    const reduceMotion = !!(config && config.reduce_motion);
    root.classList.toggle('reduce-motion', reduceMotion);
    body && body.classList.toggle('reduce-motion', reduceMotion);

    const staticOnly = !!(config && config.static_stickers_only);
    root.classList.toggle('stickers-static', staticOnly);
    body && body.classList.toggle('stickers-static', staticOnly);

    const bgUrl = config && config.custom_background_url ? config.custom_background_url.trim() : '';
    const hasBackground = !!bgUrl;
    root.classList.toggle('has-custom-bg', hasBackground);
    body && body.classList.toggle('has-custom-bg', hasBackground);
    root.style.setProperty('--personalization-bg-image', hasBackground ? `url("${bgUrl}")` : 'none');
}

function loadUIPersonalization() {
    if (isAuthPage()) {
        applyUIPersonalization({ enable_chibi_stickers: true });
        return Promise.resolve(null);
    }

    return fetch('/api/settings/personalization')
        .then(response => {
            if (response.status === 401) {
                applyUIPersonalization({ enable_chibi_stickers: true });
                return null;
            }
            return response.json();
        })
        .then(data => {
            if (!data) {
                return null;
            }
            applyUIPersonalization(data || {});
            return data;
        })
        .catch(err => {
            console.error('Failed to load UI personalization:', err);
            applyUIPersonalization({ enable_chibi_stickers: true });
            return null;
        });
}

// Snowfall animation for Christmas theme
function enableSnowfall() {
    // Remove existing snowflakes
    disableSnowfall();

    const snowContainer = document.createElement('div');
    snowContainer.id = 'snowfall-container';
    snowContainer.style.cssText = `
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        pointer-events: none;
        z-index: 9999;
        overflow: hidden;
    `;

    // Create snowflakes
    for (let i = 0; i < 50; i++) {
        createSnowflake(snowContainer);
    }

    document.body.appendChild(snowContainer);
}

function createSnowflake(container) {
    const snowflake = document.createElement('div');
    snowflake.className = 'snowflake';
    snowflake.innerHTML = '❄';

    // Random properties
    const size = Math.random() * 0.5 + 0.5; // 0.5 to 1em
    const left = Math.random() * 100; // 0 to 100%
    const animationDuration = Math.random() * 3 + 2; // 2 to 5 seconds
    const opacity = Math.random() * 0.5 + 0.3; // 0.3 to 0.8
    const delay = Math.random() * 5; // 0 to 5 seconds delay

    snowflake.style.cssText = `
        position: absolute;
        top: -10%;
        left: ${left}%;
        font-size: ${size}em;
        opacity: ${opacity};
        animation: snowfall ${animationDuration}s linear ${delay}s infinite;
        user-select: none;
    `;

    container.appendChild(snowflake);
}

function disableSnowfall() {
    const snowContainer = document.getElementById('snowfall-container');
    if (snowContainer) {
        snowContainer.remove();
    }
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    loadSavedTheme();
    loadUIPersonalization();
});
