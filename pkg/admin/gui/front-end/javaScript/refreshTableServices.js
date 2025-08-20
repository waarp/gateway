function initCollapse() {
    const container = document.getElementById('container-collapse');
    if (!container)
        return;
    const panels = Array.from(container.querySelectorAll('.no-anim-collapse'));
    const buttons = Array.from(container.querySelectorAll('button[data-bs-toggle="collapse"]'));

    panels.forEach(panel => {
        panel.classList.remove('show', 'animate');
    });
    buttons.forEach(btn => btn.setAttribute('aria-expanded', 'false'));

    panels.forEach(panel => {
        panel.addEventListener('show.bs.collapse', () => {
            panels.forEach(other => {
                if (other !== panel) other.classList.remove('show', 'animate');
            });
            panel.classList.add('animate');
            localStorage.setItem('lastOpenCollapseTasks', panel.id);

            buttons.forEach(btn => {
                const target = btn.getAttribute('data-bs-target');
                btn.setAttribute('aria-expanded', target === `#${panel.id}` ? 'true' : 'false');
            });
        });
        panel.addEventListener('hide.bs.collapse', () => {
            panel.classList.remove('animate');
        });
    });

    const last = localStorage.getItem('lastOpenCollapseTasks');
    if (last) {
        const toOpen = document.getElementById(last);
        if (toOpen) {
            toOpen.classList.add('show');
            buttons.forEach(btn => {
                const target = btn.getAttribute('data-bs-target');
                btn.setAttribute('aria-expanded', target === `#${last}` ? 'true' : 'false');
            });
        }
    }
}

function refreshServices() {
    fetch('/webui/status_services?partial=true')
        .then(response => response.text())
        .then(html => {
            document.getElementById('status').innerHTML = html;
            showSyncUpdate();
            if (window.initCollapse) {
                window.initCollapse(true);
            }
        })
        .catch(err => {
            console.error('Internal error during services refresh:', err);
        });
}

setInterval(refreshServices, 10000);
