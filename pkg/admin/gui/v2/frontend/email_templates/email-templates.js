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
    // Encodé comme un formulaire HTML (application/x-www-form-urlencoded), ce que le serveur lit ; sans le fichier.
    const params = new URLSearchParams([...new FormData(form)].filter(([, v]) => typeof v === "string"));
    fetch(form.dataset.preview, { method: "POST", body: params, credentials: "same-origin" })
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

  // Le corps depuis un fichier (#765) : le fichier choisi remplit le corps, en HTML ou en texte selon son extension, pour
  // être relu avant validation ; le champ se vide ensuite, pour que le corps relu soit celui envoyé. Sans JavaScript, le
  // fichier part avec le formulaire et le serveur le lit, avec les mêmes refus.
  const file = form.querySelector("input[type=file][name=body_file]");
  const note = document.getElementById("body_file-h");
  if (!file || !note) return;
  const say = (key, name) => { note.textContent = note.dataset[key].replace("{name}", name); };
  file.addEventListener("change", () => {
    const f = file.files && file.files[0];
    if (!f) return;
    const ext = (/\.[^.]+$/.exec(f.name.toLowerCase()) || [""])[0];
    const clear = () => { file.value = ""; file.dispatchEvent(new Event("change", { bubbles: true })); };
    if (![".html", ".htm", ".txt"].includes(ext) || f.size > 1048576) { say("refused", f.name); clear(); return; }
    f.arrayBuffer().then((buf) => {
      let body;
      try { body = new TextDecoder("utf-8", { fatal: true }).decode(buf).replace(/\r\n/g, "\n").trim(); } catch { say("refused", f.name); clear(); return; }
      const toHTML = ext !== ".txt";
      const option = [...format.options].find((o) => isHTML(o.value) === toHTML);
      if (option && format.value !== option.value) { format.value = option.value; format.dispatchEvent(new Event("change", { bubbles: true })); }
      if (toHTML) { if (html.wRte) html.wRte.load(body); else html.value = body; } else text.value = body;
      say("loaded", f.name);
      clear();
      form.dispatchEvent(new Event("input"));
    });
  });
})();
