// Transfer monitoring screen. Pages that add a transfer: the rows of the transfer information. The list: refreshes
// the states, the count and the table in place (the address keeps search, filters, state, sort and page), loads the
// state, sort and page links the same way, keeps a message when a refresh fails. Without this file, links reload.
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

  const live = document.getElementById("transfers-live");
  const errorTemplate = document.getElementById("transfers-error");
  const refreshButton = document.getElementById("transfers-refresh");
  const toolbar = document.querySelector("form.w-toolbar");
  const region = () => document.getElementById("transfers-table");
  if (!region() || !live || !errorTemplate || !window.fetch || !window.DOMParser) return;

  const period = Number(region().dataset.refreshSeconds) * 1000;
  let lastRead = live.textContent.replace(/^.*?(\d{1,2}:\d{2}:\d{2}).*$/, "$1");
  let request = null, timer = 0;

  const announce = (text) => { live.textContent = text; };
  const settled = () => announce(live.dataset.live.replace("{time}", lastRead));

  // The list as the address says, as a fragment.
  const fragmentURL = () => {
    const url = new URL(location.href);
    url.hash = "";
    url.searchParams.set("fragment", "list");
    return url;
  };

  // The menus and the list foot of the new table, as kit.js enhances them at load.
  const enhance = (root) => {
    root.querySelectorAll(".w-actions[popover]").forEach((menu) => {
      const btn = root.querySelector('[popovertarget="' + menu.id + '"]');
      const items = () => [...menu.querySelectorAll("a, button")];
      menu.addEventListener("toggle", (e) => {
        if (e.newState !== "open" || !btn) return;
        const r = btn.getBoundingClientRect();
        menu.style.position = "fixed"; menu.style.inset = "auto"; menu.style.margin = "0";
        const w = menu.offsetWidth, h = menu.offsetHeight;
        let top = r.bottom + 6;
        if (top + h > window.innerHeight - 8) top = Math.max(8, r.top - h - 6);
        menu.style.left = Math.max(8, r.right - w) + "px"; menu.style.top = top + "px";
        const first = items()[0];
        if (first) first.focus();
      });
      menu.addEventListener("keydown", (e) => {
        const list = items(), i = list.indexOf(document.activeElement);
        const map = { ArrowDown: i + 1, ArrowUp: i - 1, Home: 0, End: list.length - 1 };
        if (!(e.key in map)) return;
        e.preventDefault();
        list[(map[e.key] + list.length) % list.length].focus();
      });
      menu.addEventListener("click", (e) => { if (e.target.closest("a, button")) menu.hidePopover(); });
    });
    root.querySelectorAll(".w-pager__nav input[name=page], .w-pager__nav select[name=size]").forEach((i) => {
      i.addEventListener("change", () => {
        const page = i.name === "size" && i.form && i.form.querySelector("input[name=page]");
        if (page) page.value = 1;
        if (i.form) i.form.requestSubmit();
      });
    });
  };

  // The refresh waits while the user types in the toolbar or reads an open menu.
  const waiting = () => {
    if (document.hidden) return true;
    if (toolbar && toolbar.contains(document.activeElement)) return true;
    try { return !!document.querySelector(".w-actions[popover]:popover-open"); } catch (e) { return false; }
  };

  const showError = () => {
    const current = region();
    if (current.previousElementSibling && current.previousElementSibling.matches(".tf-refresh-error")) return;
    const banner = errorTemplate.content.firstElementChild.cloneNode(true);
    banner.classList.add("tf-refresh-error");
    banner.querySelectorAll("p").forEach((p) => {
      p.childNodes.forEach((node) => {
        if (node.nodeType === Node.TEXT_NODE) node.textContent = node.textContent.replace("{time}", lastRead);
      });
    });
    const retry = banner.querySelector("a");
    if (retry) retry.addEventListener("click", (event) => { event.preventDefault(); refresh(true); });
    current.before(banner);
  };

  // Swaps the refreshed parts, giving the focus back to the same link when it was in them.
  const swap = (fresh) => {
    const focused = document.activeElement;
    const href = focused && focused.getAttribute && focused.getAttribute("href");
    ["transfers-states", "transfers-count", "transfers-pager-top", "transfers-table"].forEach((id) => {
      const next = fresh.querySelector("#" + id), current = document.getElementById(id);
      if (next && current) current.replaceWith(next);
    });
    enhance(region());
    const top = document.getElementById("transfers-pager-top");
    if (top) enhance(top);
    if (href && !document.body.contains(focused)) {
      const again = [...document.querySelectorAll("#transfers-states a, #transfers-table a")]
        .find((a) => a.getAttribute("href") === href);
      if (again) again.focus();
    }
  };

  // A click cancels a refresh under way, a late answer never replaces a newer one; the next refresh comes a period
  // after the end of the last (2026-10-22: a slow list lost pages and jumped back).
  const later = () => { clearTimeout(timer); timer = setTimeout(() => refresh(false), period); };
  const refresh = async (asked) => {
    if (!asked && (request || waiting())) { if (!request) later(); return; }
    if (request) request.abort();
    const mine = request = new AbortController();
    region().setAttribute("aria-busy", "true");
    announce(live.dataset.refreshing);
    try {
      const response = await fetch(fragmentURL(), {
        headers: { Accept: "text/html" }, credentials: "same-origin", redirect: "error", cache: "no-store", signal: mine.signal,
      });
      if (!response.ok) throw new Error("HTTP " + response.status);
      const fresh = new DOMParser().parseFromString(await response.text(), "text/html").getElementById("transfers-fragment");
      if (!fresh) throw new Error("unexpected fragment");
      if (mine !== request) return;
      const banner = document.querySelector(".tf-refresh-error");
      if (banner) banner.remove();
      lastRead = fresh.dataset.refreshedAt || lastRead;
      swap(fresh);
    } catch (error) {
      if (mine === request) showError();
    } finally {
      if (mine === request) {
        request = null;
        const current = region();
        if (current) current.removeAttribute("aria-busy");
        settled();
        later();
      }
    }
  };

  // A state, sort or page link loads the fragment instead of the page; the address follows, so that the next
  // refreshes keep it. The menus have their own entries (pages and actions).
  document.addEventListener("click", (event) => {
    const link = event.target.closest("#transfers-states a[href], #transfers-table thead a[href], #transfers-table .w-pager a[href], #transfers-pager-top a[href]");
    if (!link || event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey) return;
    const target = new URL(link.href, location.href);
    if (target.pathname !== location.pathname) return;
    event.preventDefault();
    history.replaceState(null, "", target.search || location.pathname);
    carry(target.searchParams);
    refresh(true);
  });

  // The toolbar carries the state, the sort and the rows per page of the address, so that a new search keeps them.
  const carry = (params) => {
    if (!toolbar) return;
    ["state", "sort", "dir", "size"].forEach((name) => {
      let input = toolbar.querySelector('input[type="hidden"][name="' + name + '"]');
      if (!params.get(name)) { if (input) input.remove(); return; }
      if (!input) {
        input = document.createElement("input");
        input.type = "hidden"; input.name = name;
        toolbar.prepend(input);
      }
      input.value = params.get(name);
    });
  };

  if (refreshButton) refreshButton.addEventListener("click", (event) => { event.preventDefault(); refresh(true); });
  document.addEventListener("visibilitychange", () => { if (!document.hidden) refresh(false); });
  later();
})();
