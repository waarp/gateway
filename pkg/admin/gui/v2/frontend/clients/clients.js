// Local clients screen: how long the automatic retries take, recomputed at each keystroke (decision of 2026-10-20).
// The sentence, the timeline (SVG) and the list of the attempts under the retry fields follow the three fields: attempts, delay
// (hours, minutes, seconds) and growth factor. The computation mirrors model.RetryWaits (the pipeline's): the first
// delay, then each wait multiplied by the factor, truncated to the second, capped; an empty delay is one second.
// Without JavaScript, the sentence and the list rendered by the server stay. Texts come from the catalogue
// (data-retry, placeholders filled here).
(() => {
  "use strict";
  const root = document.querySelector("[data-retry]");
  if (!root) return;
  const t = JSON.parse(root.dataset.retry);
  const say = (key, vars) => Object.entries(vars || {}).reduce((s, [k, v]) => s.split("{" + k + "}").join(v), t[key] || key);
  const by = (id) => document.getElementById(id);
  const attempts = by("attempts"), factor = by("factor"), dh = by("delay_h"), dm = by("delay_m"), ds = by("delay_s");
  const text = root.querySelector(".w-retry__text"), svg = root.querySelector(".w-retry__svg");
  const detail = root.querySelector(".w-retry__detail"), rows = root.querySelector(".w-retry__rows");
  if (!(attempts && factor && dh && dm && ds && text && svg && detail && rows)) return;
  const NS = "http://www.w3.org/2000/svg", MAX = 2147483647, DAY = 86400, YEAR = 365;

  // The waits before each retry, as the model computes them.
  const waits = (n, first, f) => {
    n = Math.max(0, Math.min(127, n | 0));
    if (!n) return [];
    if (!(f > 0)) f = 1;
    if (first === 0) first = 1;
    const out = [];
    let cur = first;
    for (let k = 0; k < n; k++) {
      out.push(cur);
      cur = Math.min(MAX, Math.trunc(Math.fround(Math.fround(cur) * Math.fround(f))));
    }
    return out;
  };
  window.wRetryWaits = waits;   // read by the smoke test, which compares with the model

  const span = (sec) => {
    sec = Math.round(sec);
    if (sec <= 0) return say("seconds", { n: 0 });
    const days = Math.floor(sec / DAY);
    if (days >= YEAR) return say("years", { n: Math.floor(days / YEAR) });
    const rest = sec % DAY, h = Math.floor(rest / 3600), m = Math.floor((rest % 3600) / 60), s = rest % 60, p = [];
    if (days) p.push(say("days", { n: days }));
    if (h) p.push(say("hours", { n: h }));
    if (m) p.push(say("minutes", { n: m }));
    if (s) p.push(say("seconds", { n: s }));
    return p.join(" ");
  };
  const moment = (sec) => (sec >= YEAR * DAY ? span(sec) : "+" + span(sec));

  const read = () => {
    const n = parseInt(attempts.value, 10) || 0;
    let f = parseFloat((factor.value || "1").replace(",", "."));
    if (!(f > 0)) f = 1;
    const first = (parseInt(dh.value, 10) || 0) * 3600 + (parseInt(dm.value, 10) || 0) * 60 + (parseInt(ds.value, 10) || 0);
    const w = waits(n, first, f), at = [];
    let total = 0;
    w.forEach((x) => { total += x; at.push(total); });
    return { n: w.length, f, first, waits: w, at, total };
  };

  const sentence = (s) => {
    if (!s.n) return say("none");
    let out;
    if (s.n === 1) out = say("one", { delay: span(s.waits[0]) });
    else if (s.f === 1) out = say("constant", { n: s.n, delay: span(s.waits[0]), total: span(s.total) });
    else out = say("growing", { n: s.n, total: span(s.total), first: span(s.waits[0]), last: span(s.waits[s.n - 1]) });
    if (s.first === 0) out += " " + say("empty");
    return out;
  };

  const el = (tag, attrs, txt) => {
    const e = document.createElementNS(NS, tag);
    for (const k in attrs) e.setAttribute(k, attrs[k]);
    if (txt !== undefined) e.textContent = txt;
    return e;
  };

  // The timeline: a diamond for the failure, one mark per attempt placed in proportion to the elapsed time, the
  // "+duration" labels thinned when they would touch, the last one always written, the numbers below.
  const draw = (s) => {
    svg.replaceChildren();
    if (!s.n) { svg.setAttribute("hidden", ""); return; }
    svg.removeAttribute("hidden");
    const w = Math.max(320, svg.clientWidth || 640), H = 76, x0 = 30, x1 = w - 30, y = 42;
    svg.setAttribute("viewBox", "0 0 " + w + " " + H);
    const total = s.total || 1, X = (v) => x0 + ((x1 - x0) * v) / total, width = (txt) => txt.length * 6.4 + 6;
    svg.append(el("line", { class: "w-retry__axis", x1: x0, y1: y, x2: x1, y2: y }));
    svg.append(el("polygon", { class: "w-retry__origin", points: `${x0},${y - 7} ${x0 + 7},${y} ${x0},${y + 7} ${x0 - 7},${y}` }));
    svg.append(el("text", { class: "w-retry__lbl", x: x0 - 7, y: y - 14, "text-anchor": "start" }, t.failure));
    const lastText = moment(s.total), lastLeft = x1 - width(lastText), lastNumLeft = x1 - width(String(s.n)) + 4;
    let labelRight = x0 - 7 + width(t.failure), numRight = x0 - 12;
    s.at.forEach((v, k) => {
      const x = X(v), last = k === s.n - 1;
      svg.append(el("line", { class: "w-retry__tick", x1: x, y1: y - 8, x2: x, y2: y + 8 }));
      svg.append(el("circle", { class: "w-retry__mark", cx: x, cy: y, r: 5 }));
      if (last) {
        svg.append(el("text", { class: "w-retry__lbl", x: x1, y: y - 14, "text-anchor": "end" }, lastText));
      } else {
        const txt = moment(v), tw = width(txt);
        if (x - tw / 2 > labelRight + 6 && x + tw / 2 < lastLeft - 6) {
          svg.append(el("text", { class: "w-retry__lbl", x: x, y: y - 14, "text-anchor": "middle" }, txt));
          labelRight = x + tw / 2;
        }
      }
      const num = String(k + 1), nw = width(num) - 4;
      if (last || (x - nw / 2 > numRight + 4 && x + nw / 2 < lastNumLeft - 4)) {
        svg.append(el("text", { class: "w-retry__num", x: last ? x1 : x, y: y + 26, "text-anchor": last ? "end" : "middle" }, num));
        numRight = x + nw / 2;
      }
    });
  };

  // The attempts, one item each of the numbered list: the wait before it and its moment since the first failure.
  const list = (s) => {
    detail.hidden = !s.n;
    rows.replaceChildren(...s.waits.map((w, k) => {
      const li = document.createElement("li");
      li.textContent = say("row", { wait: span(w), at: moment(s.at[k]) });
      return li;
    }));
  };

  const update = () => { const s = read(); text.textContent = sentence(s); draw(s); list(s); };
  [attempts, factor, dh, dm, ds].forEach((i) => i.addEventListener("input", update));
  if ("ResizeObserver" in window) new ResizeObserver(() => draw(read())).observe(svg);
  update();
})();
