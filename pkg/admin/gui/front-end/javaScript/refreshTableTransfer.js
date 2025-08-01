function refreshTransfers() {
    if (document.querySelector('.modal.show')) {
        return;
    }
    const params = new URLSearchParams(window.location.search);
    params.set('partial', 'true');
    fetch('/webui/transfer_monitoring?' + params.toString()).then(response => response.text()).then(html => {
        document.querySelector('tbody').innerHTML = html;
    });
}

setInterval(refreshTransfers, 5000);