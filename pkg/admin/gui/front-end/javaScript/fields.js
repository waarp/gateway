function addField(button, fieldName) {
    const container = button.parentElement.querySelector(`#${fieldName.replace('[]','')}Container`);
    if (!container)
        return;

    const firstGroup = container.querySelector('.input-group');
    if (!firstGroup)
        return;
    const newGroup = firstGroup.cloneNode(true);
    const select = newGroup.querySelector('select');
    if (select)
        select.selectedIndex = 0;
    container.appendChild(newGroup);
}

function removeField(button) {
const group = button.closest('.input-group');
    const container = group.parentElement;
    const groups = container.querySelectorAll('.input-group');
    if (groups.length > 1) {
        group.remove();
    } else {
        group.querySelectorAll('input, select, textarea').forEach(el => {
            if (el.type === 'checkbox' || el.type === 'radio') {
                el.checked = false;
            } else {
                el.value = '';
            }
        });
    }
}