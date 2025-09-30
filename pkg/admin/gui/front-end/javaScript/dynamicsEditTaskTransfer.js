document.addEventListener('shown.bs.modal', function (ev) {
  const modal = ev.target;

  const send = modal.querySelector('input[name="ruleDirection"][value="send"]');
  const recv = modal.querySelector('input[name="ruleDirection"][value="receive"]');
  const ruleSel = modal.querySelector('#transferRuleSelect');

  if (send && recv && ruleSel) {
    const allSend = JSON.parse(ruleSel.getAttribute('data-rules-send') || '[]');
    const allRecv = JSON.parse(ruleSel.getAttribute('data-rules-receive') || '[]');

    function fillRules() {
      const isRecv = recv.checked;
      const rules = isRecv ? allRecv : allSend;
      const current = ruleSel.getAttribute('data-selected') || ruleSel.value || '';
      const placeholder =
        ruleSel.dataset.placeholder ||
        (ruleSel.querySelector('option[disabled]')?.textContent || 'Select a rule');

      let html = `<option value="" disabled ${current ? '' : 'selected'}>${placeholder}</option>`;
      html += rules.map(r => `<option value="${r}" ${r === current ? 'selected' : ''}>${r}</option>`).join('');
      ruleSel.innerHTML = html;
    }

    send.addEventListener('change', () => { ruleSel.setAttribute('data-selected',''); fillRules(); });
    recv.addEventListener('change', () => { ruleSel.setAttribute('data-selected',''); fillRules(); });

    fillRules();
  }

  const toSel = modal.querySelector('select[name="toTransfer"]');
  const asSel = modal.querySelector('select[name="asTransfer"]');

  if (toSel && asSel) {
    const fillAccounts = () => {
      const p = toSel.value;
      const accounts = (window.listAccountsPartner && window.listAccountsPartner[p]) || [];
      const selected = asSel.getAttribute('data-selected') || asSel.value || '';
      asSel.innerHTML =
        '<option value="" disabled>Account</option>' +
        accounts.map(a => `<option value="${a}" ${a===selected?'selected':''}>${a}</option>`).join('');
    };

    toSel.addEventListener('change', () => { asSel.setAttribute('data-selected',''); fillAccounts(); });
    fillAccounts();
  }
}, { once: false });
