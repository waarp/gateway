// The TLS variant of every protocol which has one. The TLS specific part of the
// protocol configuration is wrapped in an element whose id is the name of the
// TLS variant followed by "Form" (ex: "ftpsForm" for FTPS).
const tlsVariants = {
    'r66': 'r66-tls',
    'ftp': 'ftps',
    'http': 'https',
    'pesit': 'pesit-tls',
    'webdav': 'webdav-tls',
    'as2': 'as2-tls',
};

function setFieldsEnabled (elem, enabled) {
    elem.querySelectorAll('input,select,textarea,button').forEach(el => {
        if (enabled) {
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
}

function showProtoConfig (selectElem) {
    const selected = selectElem.value;

    const match = proto => (proto === selected || tlsVariants[proto] === selected);

    const container = selectElem.closest('.modal, form') || document;
    container.querySelectorAll('.protoConfigBlock').forEach(block => {
        const proto = block.id.replace('protoConfig_', '');
        const show = match(proto);
        block.style.display = show ? 'block' : 'none';
        setFieldsEnabled(block, show);
    });

    // The TLS sub-forms are nested inside the block of their non-TLS protocol.
    // They must thus be disabled as well as hidden when the non-TLS variant is
    // selected, otherwise their fields would still be submitted.
    Object.values(tlsVariants).forEach(tlsProto => {
        const show = selected === tlsProto;
        container.querySelectorAll(`#${tlsProto}Form`).forEach(tlsForm => {
            tlsForm.style.setProperty('display', show ? 'block' : 'none');
            setFieldsEnabled(tlsForm, show);
        });
    });
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

document.addEventListener('DOMContentLoaded', function () {
    document.querySelectorAll('.protocolConfiguration-select').forEach(sel => {
        sel.addEventListener('change', function() {
            showProtoConfig(this);
        });
        showProtoConfig(sel);
    });
});