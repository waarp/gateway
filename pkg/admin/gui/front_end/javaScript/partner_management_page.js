function tooltip() {
    const tooltipTriggerList = document.querySelectorAll('[data-bs-toggle="tooltip"]');
    const tooltipList = [...tooltipTriggerList].map(tooltipTriggerEl => new bootstrap.Tooltip(tooltipTriggerEl));
}

function showProtoConfig (selectElem) {
    const selected = selectElem.value;

    const match = proto => (
        (proto === 'r66'   && (selected === 'r66'   || selected === 'r66-tls')) ||
        (proto === 'ftp'   && (selected === 'ftp'   || selected === 'ftps')) ||
        (proto === 'pesit' && (selected === 'pesit' || selected === 'pesit-tls')) ||
        (proto === selected)
    );

    const container = selectElem.closest('.modal, form') || document;
    container.querySelectorAll('.protoConfigBlock').forEach(block => {
        const proto = block.id.replace('protoConfig_', '');
        const show = match(proto);
        block.style.display = show ? 'block' : 'none';
        block.querySelectorAll('input,select,textarea,button').forEach(el => {
            if (show) {
                el.disabled = false;
                if (el.dataset.wasRequired === '1')
                    el.required = true;
            } else {
                if (el.required)
                    el.dataset.wasRequired = '1';
                el.required = false;
                el.disabled = true;
            }
        });
    });

    container.querySelector('#ftpsForm')?.style.setProperty('display', selected === 'ftps' ? 'block' : 'none');
    container.querySelector('#pesit-tlsForm')?.style.setProperty('display', selected === 'pesit-tls' ? 'block' : 'none');
}

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
    if (container.querySelectorAll('.input-group').length > 1)
        group.remove();
}

document.addEventListener('DOMContentLoaded', function () {
    tooltip();
    document.querySelectorAll('.partner-protocol-select').forEach(sel => {
        sel.addEventListener('change', function() {
            showProtoConfig(this);
        });
        showProtoConfig(sel);
    });
});