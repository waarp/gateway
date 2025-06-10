function filterPermissions() {
    let permissions = document.getElementById('permissions');
    let permissionsType = document.getElementById('permissionsType');
    let permissionsValue = document.getElementById('permissionsValue');
    let applyBtn = document.getElementById('applyBtn');

    function changeValueSelect() {
        if ((permissions.value === "" && permissionsType.value === "" && permissionsValue.value === "") || (permissions.value && permissionsType.value && permissionsValue.value ))  {
            applyBtn.disabled = false
        } else {
            applyBtn.disabled = true
        }
    }
    permissions.addEventListener('change', changeValueSelect);
    permissionsType.addEventListener('change', changeValueSelect);
    permissionsValue.addEventListener('change', changeValueSelect);
}

function tooltip() {
    const tooltipTriggerList = document.querySelectorAll('[data-bs-toggle="tooltip"]');
    const tooltipList = [...tooltipTriggerList].map(tooltipTriggerEl => new bootstrap.Tooltip(tooltipTriggerEl));
}

function confirmPassword(password, passwordConfirm, language) {
    const pwd = document.getElementById(password);
    const confirmPwd = document.getElementById(passwordConfirm);
    if (pwd.value !== confirmPwd.value) {
        confirmPwd.classList.add('is-invalid');
        if (language === "en")
            confirmPwd.setCustomValidity('Passwords do not match');
        else if (language === "fr")
            confirmPwd.setCustomValidity('Les mots de passe ne correspondent pas');
        return false;
    } else {
        confirmPwd.classList.remove('is-invalid');
        confirmPwd.setCustomValidity('');
        return true;
    }
}

document.addEventListener('DOMContentLoaded', function () {
    filterPermissions();
    tooltip();
});