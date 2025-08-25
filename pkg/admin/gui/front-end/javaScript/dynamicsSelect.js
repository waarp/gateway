document.addEventListener('DOMContentLoaded', () => {
    const radios = document.querySelectorAll('input[name="ruleDirection"]');
    const select = document.getElementById('transferRuleSelect');
    const options = select.querySelectorAll('option');

    function filterOptions() {
        const dir = document.querySelector('input[name="ruleDirection"]:checked').value;

        options.forEach(opt => {
            if (!opt.dataset.dir)
                opt.hidden = false;
            else {
                const shouldShow = opt.dataset.dir === dir;
                opt.hidden = !shouldShow;
                if (!shouldShow && opt.selected)
                    select.value = [...options].find(o => !o.hidden).value;
            }
        });
    }

    const partnerSelect = document.getElementById('transferPartner');
    const accountSelect = document.getElementById('transferLogin');

    function updateAccounts() {
        const partner = partnerSelect.value;
        accountSelect.innerHTML = '';
        (window.listAccounts[partner] || []).forEach(name => {
            const opt = document.createElement('option');
            opt.value = name;
            opt.textContent = name;
            accountSelect.appendChild(opt);
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
    filterOptions();
    updateAccounts();
    radios.forEach(radio => radio.addEventListener('change', filterOptions));
    partnerSelect.addEventListener('change', updateAccounts);
    srcSelect.addEventListener('change', updateFilterAccounts);
});