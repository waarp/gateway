document.addEventListener('DOMContentLoaded', () => {
    const radiosTransfer = document.querySelectorAll('input[name="ruleDirection"]');
    const radiosPreRegister = document.querySelectorAll('input[name="ruleDirectionPreRegister"]');
    const transferSelect = document.getElementById('transferRuleSelect');
    const preRegisterSelect = document.getElementById('preRegisterRuleSelect');
    const transferOptions = transferSelect.querySelectorAll('option');
    const preRegisterOptions = preRegisterSelect.querySelectorAll('option');

    let transferDir = document.querySelector('input[name="ruleDirection"]:checked').value;
    radiosTransfer.forEach(radio => radio.addEventListener('change', () => {
        transferDir = document.querySelector('input[name="ruleDirection"]:checked').value;
        filterOptions(transferSelect, transferOptions, transferDir);
    }));

    let preRegisterDir = document.querySelector('input[name="ruleDirectionPreRegister"]:checked').value;
    radiosPreRegister.forEach(radio => radio.addEventListener('change', () => {
        preRegisterDir = document.querySelector('input[name="ruleDirectionPreRegister"]:checked').value;
        filterOptions(preRegisterSelect, preRegisterOptions, preRegisterDir);
    }));

    function filterOptions(selectElement, optionsElement, dirElement) {
        optionsElement.forEach(opt => {
            if (!opt.dataset.dir)
                opt.hidden = false;
            else {
                const shouldShow = opt.dataset.dir === dirElement;
                opt.hidden = !shouldShow;
                if (!shouldShow && opt.selected)
                    selectElement.value = [...optionsElement].find(o => !o.hidden).value;
            }
        });
    }

    const partnerSelect = document.getElementById('transferPartner');
    const partnerAccountSelect = document.getElementById('transferLogin');

    const serverSelect = document.getElementById('preRegisterServer');
    const serverAccountSelect = document.getElementById('preRegisterLogin');

    function updateAccounts(element, accountElement, listElement) {
        const endPoint = element.value;
        const placeholder = accountElement.querySelector('option[value=""]');
        accountElement.innerHTML = '';
        if (placeholder) {
            accountElement.appendChild(placeholder.cloneNode(true));
        }
        
        (listElement[endPoint] || []).forEach(name => {
            const opt = document.createElement('option');
            opt.value = name;
            opt.textContent = name;
            accountElement.appendChild(opt);
        });
    }

    const srcSelect = document.getElementById('filterAgent');
    const accSelect = document.getElementById('filterAccount');

    function updateFilterAccounts() {
        if (!accSelect || !srcSelect)
            return;
        accSelect.innerHTML = '';
        const placeholder = document.createElement('option');
        placeholder.value = '';
        placeholder.disabled = true;
        placeholder.textContent = accSelect.getAttribute('data-placeholder') || 'Account';
        placeholder.selected = true;
        accSelect.appendChild(placeholder);

        const key = srcSelect.value;
        if (key && window.listAgents?.[key])
            window.listAgents[key].forEach(acc => {
                const opt = document.createElement('option');
                opt.value = acc;
                opt.textContent = acc;
                if (window.filterAccountValue === acc)
                    opt.selected = true;
                accSelect.appendChild(opt);
            });
    }

    updateFilterAccounts();
    filterOptions(transferSelect, transferOptions, transferDir);
    filterOptions(preRegisterSelect, preRegisterOptions, preRegisterDir);
    updateAccounts(partnerSelect, partnerAccountSelect, window.listAccountsPartner);
    updateAccounts(serverSelect, serverAccountSelect, window.listAccountsServer);
    partnerSelect.addEventListener('change', () => {
        updateAccounts(partnerSelect, partnerAccountSelect, window.listAccountsPartner);
    });
    serverSelect.addEventListener('change', () => {
        updateAccounts(serverSelect, serverAccountSelect, window.listAccountsServer);
    });
    srcSelect.addEventListener('change', updateFilterAccounts);
});