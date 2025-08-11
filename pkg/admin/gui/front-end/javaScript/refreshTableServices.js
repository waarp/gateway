function refreshServices() {
    fetch('/webui/status_services?partial=true')
        .then(response => response.text())
        .then(html => {
            document.querySelector('#status').innerHTML = html;
            if (window.initCollapseTasks) {
                window.initCollapseTasks(true);
            }
        })
        .catch(err => {
            console.error('Internal error during services refresh:', err);
        });
}

setInterval(refreshServices, 10000);
