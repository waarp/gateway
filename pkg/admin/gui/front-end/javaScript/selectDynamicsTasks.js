document.addEventListener('DOMContentLoaded', () => {
    const container = document.getElementById('taskTRANSFER');
    if (!container)
        return;

    const radios = container.querySelectorAll('input[type="radio"][value="send"], input[type="radio"][value="receive"]');
    const ruleSelect = container.querySelector('select[name="ruleTransfer"], select#transferRuleSelect');

    if (ruleSelect) {
        const options = [...ruleSelect.querySelectorAll('option')];

        function filterOptions(dir) {
            let firstVisible = null;
            options.forEach(opt => {
                const dirAttr = opt.dataset.dir;
                const show = !dirAttr || dirAttr === dir;
                opt.hidden = !show;
                if (show && firstVisible === null)
                    firstVisible = opt;
            });

            const current = ruleSelect.value;
            const currentOpt = options.find(o => o.value === current);
            if (currentOpt && currentOpt.hidden && firstVisible) {
                ruleSelect.value = firstVisible.value;
            }
        }

        const currentDir = () => {
            const checked = [...radios].find(r => r.checked);
            return checked ? checked.value : 'send';
        };

        radios.forEach(r => r.addEventListener('change', () => filterOptions(currentDir())));
        filterOptions(currentDir());

        const modalEl = container.closest('.modal');
        if (modalEl) {
            modalEl.addEventListener('shown.bs.modal', () => {
                filterOptions(currentDir());
            });
        }
    }

    const partnerSelect = container.querySelector('select[name="toTransfer"]');
    const accountSelect = container.querySelector('#asTransfer');
    if (!(partnerSelect && accountSelect)) return;

    function populateAccountSelect(partner) {
        if (!accountSelect) return;

        const map = window.listAccountsPartner || {};
        const accounts = Array.isArray(map[partner]) ? map[partner] : [];
        [...accountSelect.querySelectorAll('option[data-dynamic="1"]')].forEach(o => o.remove());
        const existing = new Set([...accountSelect.options].map(o => o.value));

        accounts.forEach(login => {
            if (!existing.has(login)) {
                const opt = new Option(login, login);
                opt.dataset.dynamic = "1";
                accountSelect.add(opt);
            }
        });
    }

    populateAccountSelect(partnerSelect.value || '');

    partnerSelect.addEventListener('change', () => {
        populateAccountSelect(partnerSelect.value || '');
    });
});