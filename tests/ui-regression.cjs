const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');
const root = path.join(__dirname, '..');
const read = p => fs.readFileSync(path.join(root, p), 'utf8');
test('legacy stored themes cannot override the fixed internal dark theme; auth untouched', () => {
  for (const pathname of ['/', '/calendar', '/settings', '/login', '/register']) {
    for (const legacy of ['gal-violet', 'christmas', 'default', 'dark']) {
      const attrs = {}, classes = new Set();
      vm.runInNewContext(read('web/static/js/theme-init.js'), {
        window: {location: {pathname}}, localStorage: {getItem: () => legacy},
        document: {documentElement: {setAttribute: (k,v) => attrs[k]=v, classList: {add: v => classes.add(v)}}}
      });
      const auth = ['/login', '/register'].includes(pathname);
      assert.equal(attrs['data-theme'], auth ? undefined : 'dark-classic');
      assert.equal(classes.has('dark'), !auth);
    }
  }
});
test('calendar previous-month calculation handles leap February and year boundary', () => {
  const code = read('templates/calendar.html').match(/const prevMonth = (new Date\([^;]+);/)[1];
  for (const [year, month, expected] of [[2024,3,29],[2025,3,28],[2026,1,31],[2026,5,30]]) {
    assert.equal(vm.runInNewContext(code + '.getDate()', {year,month}), expected);
  }
});
test('calendar semantic surfaces retain editing and scrolling hooks', () => {
  const html = read('templates/calendar.html');
  for (const token of ['calendar-day','calendar-event','calendar-outside','calendar-date','calendar-scroll','htmx.process(cell)','hx-get="/form/subscription/${eventId}"','/api/export/ical','copyCalSubscriptionURL','PrevMonth','NextMonth','Today']) assert.ok(html.includes(token), token);
  assert.ok(!html.includes('bg-slate-50 dark:bg-slate-800/60'));
});
test('all internal pages share CSS and functioning mobile menu hooks', () => {
  for (const page of ['dashboard','subscriptions','analytics','calendar','settings']) {
    const html = read(`templates/${page}.html`);
    for (const token of ['app-ui.css','app-ui','id="mobile-menu-button"','id="mobile-menu"','mobile-menu.js','closeMobileMenu()']) assert.ok(html.includes(token), `${page}: ${token}`);
  }
});
test('theme selection and persistence removed; personalization preserved', () => {
  const settings = read('templates/settings.html'), js = read('web/static/js/themes.js');
  for (const token of ['theme-selector','selectTheme','/api/settings/theme']) assert.ok(!settings.includes(token),token);
  for (const token of ['/api/settings/theme','localStorage','enableSnowfall']) assert.ok(!js.includes(token),token);
  for (const token of ['applyUIPersonalization','loadUIPersonalization']) assert.ok(js.includes(token));
  for (const token of ['name="custom_background_url"','name="enable_chibi_stickers"','name="reduce_motion"','name="static_stickers_only"']) assert.ok(settings.includes(token));
});
