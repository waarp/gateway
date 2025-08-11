function initCollapseTasks() {
    const container = document.getElementById('container-collapse');
    if (!container)
        return;

    const panels = Array.from(container.querySelectorAll('.no-anim-collapse'));

    panels.forEach(panel =>
        bootstrap.Collapse.getOrCreateInstance(panel, { toggle: false }).hide()
    );

    panels.forEach(panel => {
        panel.removeEventListener('show.bs.collapse', panel._showListener);
        panel.removeEventListener('hide.bs.collapse', panel._hideListener);

        panel._showListener = () => {
            panels.forEach(other => {
                if (other !== panel && other.classList.contains('show')) {
                    bootstrap.Collapse.getInstance(other).hide();
                }
            });
            panel.classList.add('animate');
            localStorage.setItem('lastOpenCollapseTasks', panel.id);
        };
        panel._hideListener = () => {
            panel.classList.remove('animate');
        };

        panel.addEventListener('show.bs.collapse', panel._showListener);
        panel.addEventListener('hide.bs.collapse', panel._hideListener);
    });

    const last = localStorage.getItem('lastOpenCollapseTasks');
    if (last) {
        const toOpen = document.getElementById(last);
        if (toOpen) {
            bootstrap.Collapse.getOrCreateInstance(toOpen, { toggle: false }).show();
        }
    }
}

document.addEventListener('DOMContentLoaded', initCollapseTasks);