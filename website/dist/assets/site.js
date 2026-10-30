(() => {
  const root = document.documentElement;
  const button = document.getElementById('motion-toggle');
  const preference = matchMedia('(prefers-reduced-motion: reduce)');
  let paused = preference.matches;
  try { paused = paused || localStorage.getItem('zhile-motion-paused') === 'true'; } catch (_) {}
  const sync = () => {
    root.classList.toggle('motion-paused', paused || document.hidden || preference.matches);
    button.setAttribute('aria-pressed', String(paused || preference.matches));
    button.textContent = preference.matches ? '动效已关闭' : paused ? '开启动效' : '暂停动效';
    button.disabled = preference.matches;
  };
  button.addEventListener('click', () => {
    paused = !paused;
    try { localStorage.setItem('zhile-motion-paused', String(paused)); } catch (_) {}
    sync();
  });
  document.addEventListener('visibilitychange', sync);
  preference.addEventListener('change', sync);
  sync();
  if ('IntersectionObserver' in window && !preference.matches) {
    const observer = new IntersectionObserver(entries => {
      entries.forEach(entry => {
        if (entry.isIntersecting) { entry.target.classList.add('revealed'); observer.unobserve(entry.target); }
      });
    }, { threshold: 0.08 });
    document.querySelectorAll('.features article,.installation,.join').forEach(el => { el.classList.add('reveal'); observer.observe(el); });
  }
})();
