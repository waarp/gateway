function showSyncUpdate() {
    const syncDiv = document.getElementById('sync-update');
    const icon = syncDiv ? syncDiv.querySelector('i') : null;
    if (syncDiv) {
        syncDiv.classList.remove('d-none');
        syncDiv.style.opacity = '1';
        if (icon)
            icon.classList.add('sync-rotate');
        setTimeout(() => {
            syncDiv.style.opacity = '0';
            setTimeout(() => {
                syncDiv.classList.add('d-none');
                if (icon)
                    icon.classList.remove('sync-rotate');
            }, 300);
        }, 1000);
    }
}