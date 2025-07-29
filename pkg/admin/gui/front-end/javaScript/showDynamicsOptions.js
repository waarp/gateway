document.addEventListener('DOMContentLoaded', () => {
    const server = document.getElementById('addLocalAccountServer');
    const localAccount = document.getElementById('localAccountNameList');
    const partner = document.getElementById('addRemoteAccountPartner');
    const remoteAccount = document.getElementById('remoteAccountNameList');
    if (!server || !localAccount || !window.listLocalAccounts || !partner || !remoteAccount || !window.listRemoteAccounts)
        return;

    const localAccountBase = localAccount.innerHTML;
    const remoteAccountBase = remoteAccount.innerHTML;
    function refresh() {
        localAccount.innerHTML = localAccountBase +
        (window.listLocalAccounts[server.value] || [])
            .map(v => `<option value="${v}">${v}</option>`)
            .join('');
        remoteAccount.innerHTML = remoteAccountBase +
        (window.listRemoteAccounts[partner.value] || [])
            .map(v => `<option value="${v}">${v}</option>`)
            .join('');
    }

    server.addEventListener('change', refresh);
    partner.addEventListener('change', refresh);
    refresh();

    document.querySelectorAll('[id^="editLocalAccountServer_"]').forEach(editSrv => {
    const suffix = editSrv.id.split('_')[1];
    const editName = document.getElementById(`editLocalAccountName_${suffix}`);
    if (!editName) return;
    const baseEdit = editName.innerHTML;
    const selectedValue = editName.dataset.selected;
    editSrv.addEventListener('change', () => {
        editName.innerHTML = baseEdit +
        (window.listLocalAccounts[editSrv.value] || [])
            .map(v => `<option value="${v}" ${v === selectedValue ? 'selected' : ''}>${v}</option>`)
            .join('');
    });
    editSrv.dispatchEvent(new Event('change'));
    });

    document.querySelectorAll('[id^="editRemoteAccountPartner_"]').forEach(editSrv => {
        const suffix = editSrv.id.split('_')[1];
        const editName = document.getElementById(`editRemoteAccountName_${suffix}`);
        if (!editName) return;
        const baseEdit = editName.innerHTML;
        const selectedValue = editName.dataset.selected;
        editSrv.addEventListener('change', () => {
            editName.innerHTML = baseEdit +
              (window.listRemoteAccounts[editSrv.value] || [])
                .map(v => `<option value="${v}" ${v === selectedValue ? 'selected' : ''}>${v}</option>`)
                .join('');
            editName.value = selectedValue;
        });
        editSrv.dispatchEvent(new Event('change'));
    });
});