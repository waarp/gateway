// Service status screen: refreshes the states block every few seconds without reloading the page, loads the
// sort and page links of the tables the same way, and keeps a persistent message when a refresh fails. The page
// works without this file: every link then reloads it. Every text comes from the page (data attributes and the
// error template), none is written here.
(() => {
  "use strict";

  const block = document.getElementById("etat");
  const live = document.getElementById("services-live");
  const errorTemplate = document.getElementById("services-error");
  const refreshButton = document.getElementById("services-refresh");
  if (!block || !live || !errorTemplate || !window.fetch || !window.DOMParser) return;

  const period = Number(block.dataset.refreshSeconds) * 1000;
  let lastRead = block.dataset.refreshedAt;
  let busy = false;
  let focusAfter = false;

  const announce = (text) => { live.textContent = text; };
  const settled = () => announce(live.dataset.live.replace("{time}", lastRead));

  // The fragment of the group shown (its tile is selected by kit.js) and, when the address names that same
  // group, of the sort and the page of its table.
  const fragmentURL = () => {
    const url = new URL(block.dataset.fragmentUrl, location.href);
    const current = new URLSearchParams(location.search);
    const tab = block.querySelector('[role="tab"][aria-selected="true"]');
    url.searchParams.set("fragment", "state");
    if (tab) url.searchParams.set("group", tab.getAttribute("aria-controls"));
    if (current.get("group") === url.searchParams.get("group")) {
      ["sort", "dir", "page"].forEach((name) => { if (current.get(name)) url.searchParams.set(name, current.get(name)); });
    }
    return url;
  };

  // Refreshes the tiles and the panels in place: the elements kit.js listens to stay, the group shown stays.
  const morph = (fresh) => {
    const tabs = fresh.querySelectorAll('[role="tab"]');
    const panels = fresh.querySelectorAll('[role="tabpanel"]');
    const same = (nodes) => [...nodes].every((node) => document.getElementById(node.id));
    if (!tabs.length || !panels.length || !same(tabs) || !same(panels)) {
      block.replaceChildren(...fresh.childNodes);
      return;
    }
    tabs.forEach((tab) => {
      const current = document.getElementById(tab.id);
      current.className = tab.className;
      current.replaceChildren(...tab.childNodes);
    });
    panels.forEach((panel) => document.getElementById(panel.id).replaceChildren(...panel.childNodes));
  };

  // One message at the top of the block, naming the time of the last successful read.
  const showError = () => {
    if (block.querySelector(":scope > .w-banner")) return;
    const banner = errorTemplate.content.firstElementChild.cloneNode(true);
    banner.querySelectorAll("p").forEach((p) => {
      p.childNodes.forEach((node) => {
        if (node.nodeType === Node.TEXT_NODE) node.textContent = node.textContent.replace("{time}", lastRead);
      });
    });
    const retry = banner.querySelector("a");
    if (retry) retry.addEventListener("click", (event) => { event.preventDefault(); focusAfter = true; refresh(); });
    block.prepend(banner);
  };

  const refresh = async () => {
    if (busy) return;
    busy = true;
    block.setAttribute("aria-busy", "true");
    announce(live.dataset.refreshing);
    try {
      const response = await fetch(fragmentURL(), {
        headers: { Accept: "text/html" }, credentials: "same-origin", redirect: "error", cache: "no-store",
      });
      if (!response.ok) throw new Error("HTTP " + response.status);
      const fresh = new DOMParser().parseFromString(await response.text(), "text/html").getElementById("etat");
      if (!fresh) throw new Error("unexpected fragment");
      const banner = block.querySelector(":scope > .w-banner");
      if (banner) banner.remove();
      morph(fresh);
      lastRead = fresh.dataset.refreshedAt || lastRead;
      block.dataset.refreshedAt = lastRead;
      if (focusAfter && refreshButton) refreshButton.focus();
    } catch (error) {
      showError();
    } finally {
      focusAfter = false;
      block.removeAttribute("aria-busy");
      settled();
      busy = false;
    }
  };

  // A sort or page link of a table loads the fragment instead of the page; the address follows, so that the
  // next refreshes keep the sort and the page. The tiles (kit.js) and the retry button have their own handler.
  block.addEventListener("click", (event) => {
    const link = event.target.closest("a[href]");
    if (!link || link.getAttribute("role") === "tab" || !block.contains(link)) return;
    const target = new URL(link.href, location.href);
    if (target.pathname !== location.pathname || !target.search) return;
    event.preventDefault();
    history.replaceState(null, "", target.search + location.hash);
    refresh();
  });

  if (refreshButton) refreshButton.addEventListener("click", (event) => { event.preventDefault(); refresh(); });
  document.addEventListener("visibilitychange", () => { if (!document.hidden) refresh(); });
  setInterval(() => { if (!document.hidden) refresh(); }, period);
})();
