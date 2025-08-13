document.addEventListener('DOMContentLoaded', () => {
  const configs = [
    { methodId: 'methodEncrypt', typeId: 'keyTypeEncrypt', typesMap: window.encryptKeyTypes },
    { methodId: 'methodDecrypt', typeId: 'keyTypeDecrypt', typesMap: window.decryptKeyTypes },
    { methodId: 'methodSign',    typeId: 'keyTypeSign',    typesMap: window.signKeyTypes },
    { methodId: 'methodVerify',  typeId: 'keyTypeVerify',  typesMap: window.verifyKeyTypes },
    { methodId: 'methodEncryptSign', typeId: 'keyTypeEncryptSignEncrypt', typesMap: (window.encryptSignKeyTypes && (m => window.encryptSignKeyTypes[m]?.encrypt)) },
    { methodId: 'methodEncryptSign', typeId: 'keyTypeEncryptSignSign',    typesMap: (window.encryptSignKeyTypes && (m => window.encryptSignKeyTypes[m]?.sign)) },
    { methodId: 'methodDecryptVerify', typeId: 'keyTypeDecryptVerifyDecrypt', typesMap: (window.decryptVerifyKeyTypes && (m => window.decryptVerifyKeyTypes[m]?.decrypt)) },
    { methodId: 'methodDecryptVerify', typeId: 'keyTypeDecryptVerifyVerify',    typesMap: (window.decryptVerifyKeyTypes && (m => window.decryptVerifyKeyTypes[m]?.verify)) },
  ];

  configs.forEach(cfg => {
    const methodSel = document.getElementById(cfg.methodId);
    const typeSel   = document.getElementById(cfg.typeId);
    if (!methodSel || !typeSel) return;

    methodSel.addEventListener('change', () => {
  const placeholder = typeSel.querySelector('option');
  typeSel.innerHTML = "";
  const firstOpt = document.createElement('option');
  firstOpt.value = "";
  firstOpt.disabled = true;
  firstOpt.selected = true;
  firstOpt.textContent = placeholder ? placeholder.textContent : "";
  typeSel.appendChild(firstOpt);
      let types;
      if (typeof cfg.typesMap === 'function') {
        types = cfg.typesMap(methodSel.value) || [];
      } else {
        types = (cfg.typesMap ?? {})[methodSel.value] || [];
      }
      types.forEach(t => {
        const opt = document.createElement('option');
        opt.value = t;
        opt.textContent = t;
        typeSel.appendChild(opt);
      });
    });
  });
});