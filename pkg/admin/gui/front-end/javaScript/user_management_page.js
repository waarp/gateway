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

document.addEventListener('DOMContentLoaded', function () {
    filterPermissions();
});