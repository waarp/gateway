document.addEventListener('DOMContentLoaded', function () {
    const container = document.getElementById('tasksContainer');
    if (!container)
        return;

    const panels = Array.from(container.querySelectorAll('.no-anim-collapse'));

    panels.forEach(panel => {
        bootstrap.Collapse.getOrCreateInstance(panel, { toggle: false }).hide();
    });

    panels.forEach(panel => {
        panel.addEventListener('show.bs.collapse', () => {
            panels.forEach(other => {
                if (other !== panel && other.classList.contains('show')) {
                    bootstrap.Collapse.getInstance(other).hide();
                }
            });
            panel.classList.add('animate');
        });
        panel.addEventListener('hide.bs.collapse', () => {
            panel.classList.remove('animate');
        });
    });
});