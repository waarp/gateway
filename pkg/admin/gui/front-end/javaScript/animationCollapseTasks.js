document.addEventListener('DOMContentLoaded', function () {
    const container = document.getElementById('container-collapse');
    if (!container)
        return;

    const panels = Array.from(container.querySelectorAll('.no-anim-collapse'));

    panels.forEach(panel =>
        bootstrap.Collapse.getOrCreateInstance(panel, { toggle: false }).hide()
    );

    panels.forEach(panel => {
        panel.addEventListener('show.bs.collapse', () => {
            panels.forEach(other => {
                if (other !== panel && other.classList.contains('show')) {
                    bootstrap.Collapse.getInstance(other).hide();
                }
            });
            panel.classList.add('animate');
            localStorage.setItem('lastOpenCollapseTasks', panel.id);
        });
        panel.addEventListener('hide.bs.collapse', () => {
            panel.classList.remove('animate');
        });
    });

    const last = localStorage.getItem('lastOpenCollapseTasks');
    if (last) {
        const toOpen = document.getElementById(last);
        if (toOpen) {
            bootstrap.Collapse.getOrCreateInstance(toOpen, { toggle: false }).show();
        }
    }
});