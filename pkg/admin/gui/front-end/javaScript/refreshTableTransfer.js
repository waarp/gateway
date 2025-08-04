function disposeAllTooltips () {
    document.querySelectorAll('[data-bs-toggle="tooltip"]').forEach(el => {
        const instance = bootstrap.Tooltip.getInstance(el);
        if (instance) instance.dispose();
    });
}

function initTooltips () {
    document.querySelectorAll('[data-bs-toggle="tooltip"]').forEach(el => {
        new bootstrap.Tooltip(el);
    });
}

function refreshTransfers () {
    if (document.querySelector('.modal.show'))
        return;

    disposeAllTooltips();

    const params = new URLSearchParams(window.location.search);
    params.set('partial', 'true');
    fetch('/webui/transfer_monitoring?' + params.toString()).then(response => response.text()).then(html => {
        document.querySelector('tbody').innerHTML = html;
        initTooltips();
    })
    .catch(err => {
      console.error('Internal error during page refresh:', err);
    });
}

setInterval(refreshTransfers, 5000);
