// Transfer monitoring screen, the pages that add a transfer: the rows of the transfer information, the accounts the
// rule authorizes. The list is a live list of the kit (live.js), which refreshes it in place.
(() => {
  "use strict";

  // Transfer information: "Add" adds a row in place (without JavaScript, the form comes back with one more), a row's
  // button removes it, the rows are numbered again so that every field keeps its label.
  const infos = document.querySelector("[data-infos]");
  const rowTemplate = document.getElementById("tf-info-row");
  const addInfo = document.querySelector("[data-info-add]");
  if (infos && rowTemplate && addInfo) {
    const label = (pattern, n) => pattern.replace("{n}", n);
    const number = () => {
      infos.querySelectorAll(".tf-info-row").forEach((row, i) => {
        const n = String(i + 1);
        const [name, value] = row.querySelectorAll("input");
        const [nameLabel, valueLabel] = row.querySelectorAll("label");
        const remove = row.querySelector("[data-info-remove]");
        name.id = "info-name-" + n; value.id = "info-value-" + n;
        nameLabel.htmlFor = name.id; valueLabel.htmlFor = value.id;
        nameLabel.textContent = label(infos.dataset.labelName, n);
        valueLabel.textContent = label(infos.dataset.labelValue, n);
        remove.hidden = false;
        remove.setAttribute("aria-label", label(infos.dataset.labelRemove, n));
        remove.title = label(infos.dataset.labelRemove, n);
      });
    };
    addInfo.addEventListener("click", (event) => {
      event.preventDefault();
      infos.append(rowTemplate.content.firstElementChild.cloneNode(true));
      number();
      infos.lastElementChild.querySelector("input").focus();
    });
    infos.addEventListener("click", (event) => {
      const button = event.target.closest("[data-info-remove]");
      if (!button) return;
      const row = button.closest(".tf-info-row");
      const rows = [...infos.querySelectorAll(".tf-info-row")];
      if (rows.length === 1) {
        row.querySelectorAll("input").forEach((input) => { input.value = ""; });
        row.querySelector("input").focus();
        return;
      }
      const next = rows[rows.indexOf(row) + 1] || rows[rows.indexOf(row) - 1];
      row.remove();
      number();
      next.querySelector("input").focus();
    });
    number();
  }

  // The accounts the rule authorizes: a rule that lists who may use it (data-restricted) keeps the accounts whose
  // data-rules name it, hides the others and says how many are offered, with a link to the access of the rule; an
  // account the rule does not authorize is cleared and said. An open rule offers every account. Without JavaScript,
  // the full list, and the refusal of the server.
  const accounts = document.querySelector("select[data-rule-filter]");
  const rules = accounts && document.getElementById(accounts.dataset.ruleFilter);
  if (accounts && rules) {
    const hint = document.getElementById(accounts.id + "-h");
    const says = hint && hint.classList.contains("w-field__hint") ? hint : null;
    const base = says ? says.textContent : "";
    if (says) says.setAttribute("aria-live", "polite");
    const apply = () => {
      const rule = rules.selectedOptions[0];
      const id = rule && rule.hasAttribute("data-restricted") ? rule.value : "";
      let offered = 0;
      let cleared = "";
      accounts.querySelectorAll("option").forEach((o) => {
        if (!o.value) return;
        const ok = !id || (o.dataset.rules || "").split(" ").includes(id);
        o.hidden = !ok;
        o.disabled = !ok;
        if (ok) offered++;
        else if (o.selected) { cleared = o.textContent; accounts.value = ""; }
      });
      accounts.querySelectorAll("optgroup").forEach((g) => { g.hidden = ![...g.children].some((o) => !o.hidden); });
      if (!says) return;
      if (!id) { says.textContent = base; return; }
      const d = accounts.dataset;
      let text = offered === 0 ? d.allowedNone : offered === 1 ? d.allowedOne : d.allowed.replace("{n}", offered);
      if (cleared) text = d.cleared.replace("{login}", cleared) + " " + text;
      const link = document.createElement("a");
      link.className = "w-link";
      link.href = d.accessUrl.replace("{id}", id);
      link.textContent = d.accessLabel;
      says.replaceChildren(text + " · ", link);
    };
    rules.addEventListener("change", apply);
    apply();
  }
})();
