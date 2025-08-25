document.addEventListener('DOMContentLoaded', () => {
    const sidebarBtn = document.getElementById('sidebarCollapse');
    if (sidebarBtn)
        sidebarBtn.addEventListener('click', () => {
            sidebarBtn.blur();
        });

    const header = document.querySelector('header');
    if (header)
        document.documentElement.style.setProperty('--header-height', `${header.offsetHeight}px`);

    const store = window.localStorage;
    const getState = (key, defaultValue) => (store.getItem(key) ?? defaultValue);
    const setState = (key, value) => { try { store.setItem(key, value); } catch (_) {} };

    const applyStateSilently = (collapseEl, open) => {
        collapseEl.classList.toggle('show', open);
        const selector = `[data-bs-target="#${CSS.escape(collapseEl.id)}"]`;
        document.querySelectorAll(selector).forEach(btn => {
            btn.setAttribute('aria-expanded', String(open));
            btn.classList.toggle('collapsed', !open);
        });
      };

    const sidebar = document.getElementById('appSidebar');
    if (sidebar) {
        const sidebarStateKey = 'waarp:sb:open';
        const wantOpen = getState(sidebarStateKey, '1') === '1';
        applyStateSilently(sidebar, wantOpen);

        sidebar.addEventListener('shown.bs.collapse', () => setState(sidebarStateKey, '1'));
        sidebar.addEventListener('hidden.bs.collapse', () => setState(sidebarStateKey, '0'));

        window.addEventListener('beforeunload', () => {
            setState(sidebarStateKey, sidebar.classList.contains('show') ? '1' : '0');
        });
    }

    document.querySelectorAll('#appSidebar .collapse[id]').forEach(section => {
        const sectionStateKey = 'waarp:sb:sec:' + section.id;
        const defaultOpen = section.classList.contains('show');
        const wantOpen = getState(sectionStateKey, defaultOpen ? '1' : '0') === '1';
        applyStateSilently(section, wantOpen);

        section.addEventListener('shown.bs.collapse', () => setState(sectionStateKey, '1'));
        section.addEventListener('hidden.bs.collapse', () => setState(sectionStateKey, '0'));
        window.addEventListener('beforeunload', () => setState(sectionStateKey, section.classList.contains('show') ? '1' : '0'));
    });

    if (sidebar)
        sidebar.hidden = false;
    document.documentElement.classList.remove('sb-restoring');
    document.documentElement.classList.add('sb-animate');
    (() => {
        const currentUrl = new URL(window.location.href);
        const currentPath = currentUrl.pathname.replace(/\/+$/, '');
        const getSectionStateKey = (id) => 'waarp:sb:sec:' + id;

        document.querySelectorAll('#appSidebar .btn-toggle-nav a').forEach(link => {
            const href = link.getAttribute('href') || '';
            const url = new URL(href, window.location.href);
            const path = url.pathname.replace(/\/+$/, '');

            if (path === currentPath || currentPath.endsWith(path)) {
                link.classList.add('active');
                const section = link.closest('.collapse');
                if (!section)
                    return;
                const btn = document.querySelector(`[data-bs-target="#${section.id}"]`);
                if (btn)
                    btn.classList.add('current');
                if (!section.classList.contains('show')) {
                    bootstrap.Collapse.getOrCreateInstance(section).show();
                    try { localStorage.setItem(getSectionStateKey(section.id), '1'); } catch {}
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