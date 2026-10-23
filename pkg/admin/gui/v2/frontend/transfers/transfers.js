// Transfer monitoring screen, the pages that add a transfer: the rows of the transfer information. The list is a live
// list of the kit (live.js), which refreshes it in place.
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
})();
