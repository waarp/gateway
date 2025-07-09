function noSelectedPartner(errMsg) {
    alert(errMsg);
    window.location = "partner_management";
}

function tooltip() {
    const tooltipTriggerList = document.querySelectorAll('[data-bs-toggle="tooltip"]');
    const tooltipList = [...tooltipTriggerList].map(tooltipTriggerEl => new bootstrap.Tooltip(tooltipTriggerEl));
}

function showCredentialTypeBlock(selectElem) {
    const val = selectElem.value;

    if (selectElem.classList.contains('add-credential-type-select')) {
        const passwordBlock = document.getElementById('addPasswordType');
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
        const id = modal.id.replace('editCredentialPartnerModal_', '');
        const passwordBlock = document.getElementById('editPasswordType_' + id);
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

function readFile(hiddenName, fileName) {
    const file = document.querySelector('input[type="file"][name="' + fileName + '"]');
    const hidden = document.getElementById(hiddenName);
    if (file && hidden) {
        file.addEventListener('change', e => {
            const file = e.target.files[0];
            if (!file) return hidden.value = "";
            const reader = new FileReader();
            reader.onload = event => hidden.value = event.target.result;
            reader.readAsText(file);
        });
    }
}

document.addEventListener('DOMContentLoaded', function () {
    tooltip();
    readFile('addCredentialPartnerValueFile', 'addCredentialPartnerFile');
    document.querySelectorAll('.add-credential-type-select, .edit-credential-type-select').forEach(sel => {
        sel.addEventListener('change', function() {
            showCredentialTypeBlock(this);
        });
        showCredentialTypeBlock(sel);
    });
});