// Email templates screen (#744): the preview of the form follows the typing, and the body follows the format. Native
// JavaScript, no dependency; without it the form works and the preview shows the template as the page loaded it.
(() => {
  "use strict";

  const form = document.querySelector("form[data-preview]");
  const out = document.getElementById("et-preview-out");
  if (!form || !out) return;

  // L'aperçu : le formulaire est envoyé au serveur, qui substitue les marqueurs avec le code de la tâche EMAIL ; une
  // réponse plus ancienne que la dernière demande est ignorée.
  let timer = 0;
  let asked = 0;
  const refresh = () => {
    const n = ++asked;
    fetch(form.dataset.preview, { method: "POST", body: new FormData(form), credentials: "same-origin" })
      .then((r) => (r.ok ? r.text() : Promise.reject(new Error(String(r.status)))))
      .then((html) => { if (n === asked) out.innerHTML = html; })
      .catch(() => {});
  };
  const schedule = () => { clearTimeout(timer); timer = setTimeout(refresh, 300); };
  form.addEventListener("input", schedule);
  form.addEventListener("change", schedule);

  // Le format : le corps passe d'un champ à l'autre, du texte en paragraphes, du HTML en texte.
  const format = form.querySelector("select[name=format]");
  const text = form.querySelector("textarea[name=body_text]");
  const html = form.querySelector("textarea[name=body_html]");
  if (!format || !text || !html) return;
  const isHTML = (v) => v.split(";")[0].trim() === "text/html";
  let wasHTML = isHTML(format.value);
  const escape = (s) => s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
  format.addEventListener("change", () => {
    const nowHTML = isHTML(format.value);
    if (nowHTML && !wasHTML && text.value.trim()) {
      const body = text.value.split(/\n{2,}/).filter((p) => p.trim())
        .map((p) => "<p>" + escape(p).replace(/\n/g, "<br>") + "</p>").join("");
      if (html.wRte) html.wRte.load(body); else html.value = body;
    } else if (!nowHTML && wasHTML && html.value.trim()) {
      const doc = new DOMParser().parseFromString(html.value, "text/html");
      doc.querySelectorAll("p, h2, h3, li, br").forEach((n) => n.after(n.tagName === "BR" ? "\n" : "\n\n"));
      text.value = doc.body.textContent.replace(/​/g, "").replace(/\n{3,}/g, "\n\n").trim();
    }
    wasHTML = nowHTML;
  });
})();
