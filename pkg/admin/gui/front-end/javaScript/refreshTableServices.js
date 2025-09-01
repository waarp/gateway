function initCollapses(root = document) {
    root.querySelectorAll('#status .collapse').forEach(el => {
        try {
            bootstrap.Collapse.getOrCreateInstance(el, { toggle: false });
        } catch (e) {
            console.error('Bootstrap Collapse init error:', e);
        }
    });
}

function refreshServices() {
    const openIds = Array.from(document.querySelectorAll('#status .collapse.show')).map(el => el.id);

    fetch('/webui/status_services?partial=true')
        .then(response => response.text())
        .then(html => {
            const container = document.getElementById('status');
            if (!container)
                return;
            container.innerHTML = html;
            initCollapses(container);
            openIds.forEach(id => {
                const el = container.querySelector('#' + CSS.escape(id));
                if (el) {
                    try { bootstrap.Collapse.getOrCreateInstance(el, { toggle: false }).show(); } catch (_) {}
                }
            });
            showSyncUpdate();
        })
        .catch(err => {
            console.error('Internal error during services refresh:', err);
        });
}

setInterval(refreshServices, 10000);
document.addEventListener('DOMContentLoaded', () => initCollapses());
