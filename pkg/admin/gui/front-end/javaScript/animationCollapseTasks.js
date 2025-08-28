document.addEventListener('DOMContentLoaded', () => {
    const collapseContainer = document.getElementById('container-collapse');
    if (!collapseContainer)
        return;

    const collapsePanels = Array.from(collapseContainer.querySelectorAll('.no-anim-collapse'));

    collapsePanels.forEach(panel =>
        bootstrap.Collapse.getOrCreateInstance(panel, { toggle: false }).hide()
    );

    collapsePanels.forEach(panel => {
        panel.addEventListener('show.bs.collapse', () => {
            collapsePanels.forEach(otherPanel => {
                if (otherPanel !== panel && otherPanel.classList.contains('show')) {
                    bootstrap.Collapse.getInstance(otherPanel).hide();
                }
            });
            panel.classList.add('animate');
            localStorage.setItem('lastOpenCollapseTasks', panel.id);
        });
        panel.addEventListener('hide.bs.collapse', () => {
            panel.classList.remove('animate');
        });
    });

    const lastOpenPanelId = localStorage.getItem('lastOpenCollapseTasks');
    if (lastOpenPanelId) {
        const lastOpenPanel = document.getElementById(lastOpenPanelId);
        if (lastOpenPanel) {
            bootstrap.Collapse.getOrCreateInstance(lastOpenPanel, { toggle: false }).show();
        }
    }
});