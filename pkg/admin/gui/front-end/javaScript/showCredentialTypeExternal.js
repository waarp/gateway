function showCredentialTypeBlock(selectElem) {
    const val = selectElem.value;

    if (selectElem.classList.contains('add-credential-type-select')) {
        const passwordBlock = document.getElementById('addTextType');
        const fileBlock = document.getElementById('addFileType');
        const fileAndPwdBlock = document.getElementById('addTwoFileType');
        const loginAndPwdBlock = document.getElementById('addLoginAndPwdType');
        if (passwordBlock) passwordBlock.style.display = 'none';
        if (fileBlock) fileBlock.style.display = 'none';
        if (fileAndPwdBlock) fileAndPwdBlock.style.display = 'none';
        if (loginAndPwdBlock) loginAndPwdBlock.style.display = 'none';

        if (val === 'password') {
            if (passwordBlock) passwordBlock.style.display = 'block';
        } else if (val === 'ssh_private_key') {
            if (fileBlock) fileBlock.style.display = 'block';
        } else if (val === 'tls_certificate') {
            if (fileAndPwdBlock) fileAndPwdBlock.style.display = 'block';
        } else if (val === 'pesit_pre-connection_auth') {
            if (loginAndPwdBlock) loginAndPwdBlock.style.display = 'block';
        }
    }

    if (selectElem.classList.contains('edit-credential-type-select')) {
        const modal = selectElem.closest('.modal');
        if (!modal) return;
        const id = modal.id.replace('editCredentialPartnerModal_', '').replace('editCredentialAccountModal_', '');
        const passwordBlock = document.getElementById('editTextType_' + id);
        const fileBlock = document.getElementById('editFileType_' + id);
        const fileAndPwdBlock = document.getElementById('editTwoFileType_' + id);
        const loginAndPwdBlock = document.getElementById('editLoginAndPwdType_' + id);
        if (passwordBlock) passwordBlock.style.display = 'none';
        if (fileBlock) fileBlock.style.display = 'none';
        if (fileAndPwdBlock) fileAndPwdBlock.style.display = 'none';
        if (loginAndPwdBlock) loginAndPwdBlock.style.display = 'none';

        if (val === 'password') {
            if (passwordBlock) passwordBlock.style.display = 'block';
        } else if (val === 'ssh_private_key') {
            if (fileBlock) fileBlock.style.display = 'block';
        } else if (val === 'tls_certificate') {
            if (fileAndPwdBlock) fileAndPwdBlock.style.display = 'block';
        } else if (val === 'pesit_pre-connection_auth') {
            if (loginAndPwdBlock) loginAndPwdBlock.style.display = 'block';
        }
    }
}

document.addEventListener('DOMContentLoaded', function () {
    readFile('addCredentialValueFile', 'addCredentialFile');
    readFile('addCredentialValueFile1', 'addCredentialFile1');
    readFile('addCredentialValueFile2', 'addCredentialFile2');
    readFile('editCredentialValueFile', 'editCredentialFile');
    readFile('editCredentialValueFile1', 'editCredentialFile1');
    readFile('editCredentialValueFile2', 'editCredentialFile2');
    document.querySelectorAll('.add-credential-type-select, .edit-credential-type-select').forEach(sel => {
        sel.addEventListener('change', function() {
            showCredentialTypeBlock(this);
        });
        showCredentialTypeBlock(sel);
    });
});