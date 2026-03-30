(function () {
    const base = '/static/images/chibi/';

    const stickers = {
        normal: { key: 'chibi-red-envelope', label: 'Normal' },
        chargeSoon: { key: 'chibi-squish-cry', label: 'Charge Soon' },
        lowUse: { key: 'chibi-peek', label: 'Low Use' },
        reminderOn: { key: 'chibi-cat-ears', label: 'Set Reminder' },
        reminderOff: { key: 'chibi-peek', label: 'Reminder Off' },
        monthlySpend: { key: 'chibi-teary', label: 'This Month' },
        growth: { key: 'chibi-coin', label: 'Growth' },
        savings: { key: 'chibi-red-envelope', label: 'Savings' }
    };

    function asset(key, animated) {
        return base + key + (animated ? '.png' : '-still.png');
    }

    function markMissing(img) {
        if (!img) return;
        img.style.display = 'none';
        const pair = img.closest('.sticker-pair');
        if (pair) {
            pair.classList.add('sticker-missing');
        }
    }

    function wireFallbacks(scope) {
        const root = scope || document;
        root.querySelectorAll('.sticker-pair img').forEach(function (img) {
            if (img.dataset.stickerBound === 'true') return;
            img.dataset.stickerBound = 'true';
            img.addEventListener('error', function () {
                markMissing(img);
            });
        });
    }

    function applyStickerPair(target, key) {
        if (!target) return;
        const still = target.querySelector('.sticker-still');
        const animated = target.querySelector('.sticker-animated');
        if (still) still.src = asset(key, false);
        if (animated) animated.src = asset(key, true);
        target.classList.remove('sticker-missing');
        wireFallbacks(target);
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', function () {
            wireFallbacks(document);
        });
    } else {
        wireFallbacks(document);
    }

    window.SubTrackrStickers = {
        sizes: {
            small: 36,
            card: 56,
            hero: 88
        },
        stickers,
        asset,
        applyStickerPair,
        wireFallbacks
    };
})();
