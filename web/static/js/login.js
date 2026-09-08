// Spotlight coordinate technique adapted from React Bits SpotlightCard by David Haz.
// https://github.com/DavidHDev/react-bits/blob/main/src/content/Components/SpotlightCard/SpotlightCard.jsx
// See /static/licenses/react-bits-spotlight.md. Native DOM only; no React runtime.
(() => {
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
    document.querySelectorAll('[data-spotlight]').forEach(field => {
        field.addEventListener('pointermove', event => {
            if (reducedMotion.matches || event.pointerType === 'touch') return;
            const rect = field.getBoundingClientRect();
            field.style.setProperty('--mouse-x', `${event.clientX - rect.left}px`);
            field.style.setProperty('--mouse-y', `${event.clientY - rect.top}px`);
        });
    });
    const reveal = document.querySelector('.signin-reveal');
    const password = document.getElementById('password');
    reveal.addEventListener('click', () => {
        const visible = password.type === 'password';
        password.type = visible ? 'text' : 'password';
        reveal.setAttribute('aria-pressed', String(visible));
        reveal.setAttribute('aria-label', visible ? '隐藏密码' : '显示密码');
    });
})();
