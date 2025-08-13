function disposeAllTooltips () {
    document.querySelectorAll('[data-bs-toggle="tooltip"], button[title]').forEach(el => {
        const instance = bootstrap.Tooltip.getInstance(el);
        if (instance) instance.dispose();
    });
}

function initTooltips () {
    document.querySelectorAll('[data-bs-toggle="tooltip"]').forEach(el => {
        new bootstrap.Tooltip(el);
    });
    document.querySelectorAll('button[title]').forEach(el => {
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
        document.querySelectorAll('input.form-control-plaintext, input.form-control-plaintext[readonly], input.form-control-plaintext[disabled]').forEach(function(input) {
            if (input.value === '<non définie>' || input.value === '<undefined>') {
                input.classList.add('input-undefined');
            }
        });
    })
    .catch(err => {
      console.error('Internal error during page refresh:', err);
    });
}

setInterval(refreshTransfers, 5000);
