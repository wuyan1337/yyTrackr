// One internal theme, established before paint. Auth keeps its dedicated styling.
(function() {
    const path = window.location.pathname.replace(/\/$/, '') || '/';
    if (['/login', '/register', '/forgot-password', '/reset-password'].includes(path)) return;
    document.documentElement.setAttribute('data-theme', 'dark-classic');
    document.documentElement.classList.add('dark');
})();
