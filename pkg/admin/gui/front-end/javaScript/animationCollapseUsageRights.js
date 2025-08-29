document.addEventListener('DOMContentLoaded', () => {
    const usageRightsContainer = document.getElementById('container-usage-rights');
    if (!usageRightsContainer) 
        return;

    const usageRightsPanels = Array.from(usageRightsContainer.querySelectorAll('.no-anim-collapse'));

    usageRightsPanels.forEach(panel =>
        bootstrap.Collapse.getOrCreateInstance(panel, { toggle: false }).hide()
    );

    usageRightsPanels.forEach(panel => {
        panel.addEventListener('show.bs.collapse', () => {
            panel.classList.add('animate');
            localStorage.setItem('lastOpenCollapseUsageRights', panel.id);
        });
        panel.addEventListener('hide.bs.collapse', () => {
            panel.classList.remove('animate');
        });
    });

    const lastOpenPanelId = localStorage.getItem('lastOpenCollapseUsageRights');
    if (lastOpenPanelId) {
        const lastOpenPanel = document.getElementById(lastOpenPanelId);
        if (lastOpenPanel) {
            bootstrap.Collapse.getOrCreateInstance(lastOpenPanel, { toggle: false }).show();
        }
    }
});