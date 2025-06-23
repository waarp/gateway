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

function addField(btn, inputName) {
    const container = btn.closest('.form-input').querySelector('div[id$="Container"]');
    if (!container) return;
    const group = document.createElement('div');
    group.className = 'input-group mb-2';
    group.innerHTML = `
        <input type="text" name="${inputName}" class="form-control" required>
        <button type="button" class="btn btn-outline-danger btn-sm" onclick="removeField(this)" tabindex="-1">
            <i class="bi bi-trash"></i>
        </button>
    `;
    container.appendChild(group);
}

function removeField(btn) {
    const group = btn.closest('.input-group');
    if (!group) {
        return;
    }
    const container = group.parentElement;
    if (container.querySelectorAll('.input-group').length > 1) {
        group.remove();
    }
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