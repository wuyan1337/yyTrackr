// The internal theme is fixed by theme-init.js; only UI personalization is persisted.
function isAuthPage() {
    const path = window.location.pathname.replace(/\/$/, '');
    return ['/login', '/register', '/forgot-password', '/reset-password'].includes(path);
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
            if (!data) return null;
            applyUIPersonalization(data || {});
            return data;
        })
        .catch(err => {
            console.error('Failed to load UI personalization:', err);
            applyUIPersonalization({ enable_chibi_stickers: true });
            return null;
        });
}

document.addEventListener('DOMContentLoaded', () => {
    loadUIPersonalization();
});
