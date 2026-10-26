// Dashboard screen: refreshes the figures every half minute without reloading the page, on the "Refresh" button
// and when the tab comes back to the front, and keeps a persistent message when a refresh fails. The parts of the
// body are replaced one by one, so that the open or closed state of the values section and the listeners of the
// kit survive; the actions menus of the new rows are handed to the kit. The page works without this file: the
// button then reloads it. Every text comes from the page (data attributes and the error template), none is
// written here.
(() => {
  "use strict";

  const block = document.getElementById("tdb-body");
  const live = document.getElementById("tdb-live");
  const errorTemplate = document.getElementById("tdb-error");
  const refreshButton = document.getElementById("tdb-refresh");
  if (!block || !live || !errorTemplate || !window.fetch || !window.DOMParser) return;

  const period = Number(block.dataset.refreshSeconds) * 1000;
  let lastRead = block.dataset.refreshedAt;
  let busy = false;
  let focusAfter = false;

  const announce = (text) => { live.textContent = text; };
  const settled = () => announce(live.dataset.live.replace("{time}", lastRead));

  const fragmentURL = () => {
    const url = new URL(block.dataset.fragmentUrl, location.href);
    url.searchParams.set("fragment", "body");
    return url;
  };

  // Replaces the content of every part of the body by its fresh version; a part the page did not have, or lost,
  // makes the whole body replaced. The menus of the new rows are enhanced by the kit; the tables that overflow
  // are made focusable again (the kit does it on a resize).
  const morph = (fresh) => {
    const parts = [...fresh.querySelectorAll("[data-tdb-part]")];
    const same = parts.length > 0 && parts.every((part) => document.getElementById(part.id))
      && parts.length === block.querySelectorAll("[data-tdb-part]").length;
    if (!same) {
      block.replaceChildren(...fresh.childNodes);
    } else {
      parts.forEach((part) => document.getElementById(part.id).replaceChildren(...part.childNodes));
    }
    if (window.wEnhance) window.wEnhance(block);
    window.dispatchEvent(new Event("resize"));
  };

  // One message at the top of the body, naming the time of the last successful read.
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
      const fresh = new DOMParser().parseFromString(await response.text(), "text/html").getElementById("tdb-body");
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

  if (refreshButton) refreshButton.addEventListener("click", (event) => { event.preventDefault(); refresh(); });
  document.addEventListener("visibilitychange", () => { if (!document.hidden) refresh(); });
  setInterval(() => { if (!document.hidden) refresh(); }, period);
})();
