function showSyncUpdate() {
    const syncDiv = document.getElementById('sync-update');
    if (syncDiv) {
        syncDiv.classList.remove('d-none');
        syncDiv.style.opacity = '1';
        setTimeout(() => {
            syncDiv.style.opacity = '0';
            setTimeout(() => syncDiv.classList.add('d-none'), 300);
        }, 1000);
    }
}