document.addEventListener('DOMContentLoaded', () => {
  const sidebarBtn = document.getElementById('sidebarCollapse');
  if (sidebarBtn) {
    sidebarBtn.addEventListener('click', () => {
      sidebarBtn.blur();
    });
  }
  const header = document.querySelector('header');
  if (header)
    document.documentElement.style.setProperty('--header-height', `${header.offsetHeight}px`);

  const store = window.localStorage;
  const get = (k, d) => (store.getItem(k) ?? d);
  const set = (k, v) => { try { store.setItem(k, v); } catch (_) {} };

  const applyStateSilently = (collapseEl, open) => {
    collapseEl.classList.toggle('show', open);
    const sel = `[data-bs-target="#${CSS.escape(collapseEl.id)}"]`;
    document.querySelectorAll(sel).forEach(btn => {
      btn.setAttribute('aria-expanded', String(open));
      btn.classList.toggle('collapsed', !open);
    });
  };

  const wrap = document.getElementById('appSidebar');
  if (wrap) {
    const K = 'waarp:sb:open';
    const wantOpen = get(K, '1') === '1';
    applyStateSilently(wrap, wantOpen);

    wrap.addEventListener('shown.bs.collapse', () => set(K, '1'));
    wrap.addEventListener('hidden.bs.collapse', () => set(K, '0'));

    window.addEventListener('beforeunload', () => {
      set(K, wrap.classList.contains('show') ? '1' : '0');
    });
  }

  document.querySelectorAll('#appSidebar .collapse[id]').forEach(sec => {
    const key = 'waarp:sb:sec:' + sec.id;
    const defaultOpen = sec.classList.contains('show');
    const wantOpen = get(key, defaultOpen ? '1' : '0') === '1';
    applyStateSilently(sec, wantOpen);

    sec.addEventListener('shown.bs.collapse', () => set(key, '1'));
    sec.addEventListener('hidden.bs.collapse', () => set(key, '0'));
    window.addEventListener('beforeunload', () => set(key, sec.classList.contains('show') ? '1' : '0'));
  });

  document.documentElement.classList.remove('sb-restoring');
  (() => {
    const here = new URL(window.location.href);
    const herePath = here.pathname.replace(/\/+$/, '');
    const storeKey = (id) => 'waarp:sb:sec:' + id;

    document.querySelectorAll('#appSidebar .btn-toggle-nav a').forEach(a => {
      const href = a.getAttribute('href') || '';
      const url  = new URL(href, window.location.href);
      const path = url.pathname.replace(/\/+$/, '');

      if (path === herePath || herePath.endsWith(path)) {
        a.classList.add('active');
        const sec = a.closest('.collapse');
        if (!sec) 
          return;
        const btn = document.querySelector(`[data-bs-target="#${sec.id}"]`);
        if (btn)
          btn.classList.add('current');
        if (!sec.classList.contains('show')) {
          bootstrap.Collapse.getOrCreateInstance(sec).show();
          try { localStorage.setItem(storeKey(sec.id), '1'); } catch {}
        }
      }
    });
  })();
});

document.querySelectorAll('.btn-toggle').forEach(btn => {
  btn.addEventListener('click', () => {
    btn.blur();
  });
});
