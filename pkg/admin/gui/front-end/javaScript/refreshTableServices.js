const refreshID = setInterval(refreshServices, 10000);
document.addEventListener('DOMContentLoaded', () => initCollapses());

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
    const request = new Request('/webui/status_services?partial=true', {redirect: 'error'})

    fetch(request)
        .then(resp => {
            if (!resp.ok) return Promise.reject(resp)
            return resp.text()
        })
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
            clearInterval(refreshID);
            console.error('Internal error during services refresh:', err);
        });
}

