function addFieldCloudOptions(button) {
    let container = button.parentElement.querySelector('#infoContainer');
    if (!container)
        container = button.parentElement.querySelector('[id^="editInfoContainer_"]');
    if (!container)
        return;

    const firstGroup = container.querySelector('.input-group');
    if (!firstGroup)
        return;
    const newGroup = firstGroup.cloneNode(true);

    newGroup.querySelectorAll('input').forEach(el => el.value = '');

    container.appendChild(newGroup);
}

function removeFieldCloudOptions(button) {
    const group = button.closest('.input-group');
    const container = group.parentElement;
    const groups = container.querySelectorAll('.input-group');
    if (groups.length > 1)
        group.remove();
    else
        group.querySelectorAll('input, select, textarea').forEach(el => {
            if (el.type === 'checkbox' || el.type === 'radio')
                el.checked = false;
            else
                el.value = '';
        });
}