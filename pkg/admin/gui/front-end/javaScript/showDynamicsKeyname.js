function parseAllowed(keysString) {
	if (!keysString)
		return [];
	return keysString.split(",").map(s => s.trim()).filter(Boolean);
}

function setHidden(element, hide) {
	if (!element)
		return;
	if (hide) {
		element.setAttribute("hidden", "");
		element.style.display = "none";
	} else {
		element.removeAttribute("hidden");
		element.style.display = "";
	}
}

function filterSelectByAllowed(selectElement, allowed) {
	if (!selectElement)
		return;
	const groups = Array.from(selectElement.querySelectorAll("optgroup"));
	let hasVisible = false;

	groups.forEach(g => {
		const type = g.dataset.type || g.getAttribute("label");
		const show = allowed.includes(type);
		setHidden(g, !show);

		const opts = g.querySelectorAll("option");
		opts.forEach(o => {
		setHidden(o, !show);
		o.disabled = !show;
		});

		if (show && opts.length > 0)
		hasVisible = true;
	});

	const directOptions = Array.from(selectElement.children).filter(n => n.tagName === "OPTION");
	directOptions.forEach(o => {
		const isPlaceholder = o.value === "" || o.disabled;
		if (isPlaceholder)
		return;

		setHidden(o, true);
		o.disabled = true;
	});

	const sel = selectElement.selectedOptions && selectElement.selectedOptions[0];
	if (sel && (sel.disabled || sel.hidden || sel.style.display === "none")) {
		selectElement.selectedIndex = 0;
	}

	selectElement.disabled = allowed.length === 0 || !hasVisible;
}

function attachKeyFilter(methodSelectId, keySelectId) {
	const method = document.getElementById(methodSelectId);
	const keySel = document.getElementById(keySelectId);
	if (!method || !keySel)
		return;

	const update = () => {
		const opt = method.options[method.selectedIndex];
		const allowed = parseAllowed(opt && opt.dataset.keytypes);
		filterSelectByAllowed(keySel, allowed);
	};

	method.addEventListener("change", update);
	update();
}

function attachDualKeyFilter(methodSelectId, mappings) {
	const method = document.getElementById(methodSelectId);
	if (!method)
		return;

	const cache = {};
	mappings.forEach(m => {
		cache[m.selectId] = document.getElementById(m.selectId);
	});

	const update = () => {
		const opt = method.options[method.selectedIndex];
		mappings.forEach(m => {
		const allowedRaw = opt ? opt.dataset[m.datasetKey] : "";
		const allowed = parseAllowed(allowedRaw);
		filterSelectByAllowed(cache[m.selectId], allowed);
		});
	};

	method.addEventListener("change", update);
	update();
}

document.addEventListener("DOMContentLoaded", function () {
	attachKeyFilter("methodEncryptAdd", "keyNameEncryptAdd");
	attachKeyFilter("methodEncryptEdit", "keyNameEncryptEdit");

	attachKeyFilter("methodDecryptAdd", "keyNameDecryptAdd");
	attachKeyFilter("methodDecryptEdit", "keyNameDecryptEdit");

	attachKeyFilter("methodSignAdd", "keyNameSignAdd");
	attachKeyFilter("methodSignEdit", "keyNameSignEdit");

	attachKeyFilter("methodVerifyAdd", "keyNameVerifyAdd");
	attachKeyFilter("methodVerifyEdit", "keyNameVerifyEdit");

	attachDualKeyFilter("methodEncryptSignAdd", [
		{ selectId: "encryptKeyNameEncrypt&SignAdd", datasetKey: "keytypesEncrypt" },
		{ selectId: "signKeyNameEncrypt&SignAdd", datasetKey: "keytypesSign" },
	]);

	attachDualKeyFilter("methodEncryptSignEdit", [
		{ selectId: "encryptKeyNameEncrypt&SignEdit", datasetKey: "keytypesEncrypt" },
		{ selectId: "signKeyNameEncrypt&SignEdit", datasetKey: "keytypesSign" },
	]);

	attachDualKeyFilter("methodDecryptVerifyAdd", [
		{ selectId: "decryptKeyNameDecrypt&verifyAdd", datasetKey: "keytypesDecrypt" },
		{ selectId: "verifyKeyNameDecrypt&verifyAdd", datasetKey: "keytypesVerify" },
	]);

	attachDualKeyFilter("methodDecryptVerifyEdit", [
		{ selectId: "decryptKeyNameDecrypt&verifyEdit", datasetKey: "keytypesDecrypt" },
		{ selectId: "verifyKeyNameDecrypt&verifyEdit", datasetKey: "keytypesVerify" },
	]);
});