function showCredentialTypeBlock(selectElem) {
    const val = selectElem.value;

    if (selectElem.classList.contains('add-credential-type-select')) {
        const passwordBlock = document.getElementById('addTextType');
        const fileBlock = document.getElementById('addFileType');
        if (passwordBlock) passwordBlock.style.display = 'none';
        if (fileBlock) fileBlock.style.display = 'none';
        if (val === 'password') {
            if (passwordBlock) passwordBlock.style.display = 'block';
        } else if (val === 'trusted_tls_certificate' || val === 'ssh_public_key') {
            if (fileBlock) fileBlock.style.display = 'block';
        }
    }

    if (selectElem.classList.contains('edit-credential-type-select')) {
        const modal = selectElem.closest('.modal');
        if (!modal) return;
        const id = modal.id.replace('editCredentialInternalModal_', '');
        const passwordBlock = document.getElementById('editTextType_' + id);
        const fileBlock = document.getElementById('editFileType_' + id);
        if (passwordBlock) passwordBlock.style.display = 'none';
        if (fileBlock) fileBlock.style.display = 'none';
        if (val === 'password') {
            if (passwordBlock) passwordBlock.style.display = 'block';
        } else if (val === 'trusted_tls_certificate' || val === 'ssh_public_key') {
            if (fileBlock) fileBlock.style.display = 'block';
        }
    }
}

document.addEventListener('DOMContentLoaded', function () {
    readFile('addCredentialValueFile', 'addCredentialFile');
    readFile('editCredentialValueFile', 'editCredentialFile');

    document.querySelectorAll('.add-credential-type-select, .edit-credential-type-select').forEach(sel => {
        sel.addEventListener('change', function() {
            showCredentialTypeBlock(this);
        });
        showCredentialTypeBlock(sel);
    });
});