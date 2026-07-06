const tocLinks = Array.from(document.querySelectorAll("[data-toc-link]"));

if (tocLinks.length > 0) {
  const targetIds = Array.from(new Set(tocLinks.map((link) => link.getAttribute("data-toc-target")).filter(Boolean)));
  const entries = targetIds.map((id) => {
    const target = document.getElementById(id);
    if (!target) {
      return undefined;
    }
    const links = tocLinks.filter((link) => link.getAttribute("data-toc-target") === id);
    const markerItems = links.map((link) => link.closest("[data-toc-marker-item]")).filter(Boolean);
    return {
      id,
      links,
      markerItems,
      progress: 0,
      scrollTarget: 0,
      target,
    };
  }).filter(Boolean);
  let activeId = "";
  let updateFrame;

  const clamp = (value, min, max) => Math.min(max, Math.max(min, value));

  const maxScrollY = () => Math.max(0, document.documentElement.scrollHeight - window.innerHeight);

  const documentTop = (element) => element.getBoundingClientRect().top + window.scrollY;

  const scrollMarginTop = (element) => {
    const value = window.getComputedStyle(element).scrollMarginTop;
    const parsed = Number.parseFloat(value);
    return Number.isFinite(parsed) ? parsed : 0;
  };

  const updateTargetPositions = () => {
    const maxScroll = maxScrollY();
    for (const [index, entry] of entries.entries()) {
      entry.scrollTarget = clamp(documentTop(entry.target) - scrollMarginTop(entry.target), 0, maxScroll);
      entry.progress = maxScroll > 0
        ? entry.scrollTarget / maxScroll
        : entries.length > 1 ? index / (entries.length - 1) : 0;
      for (const markerItem of entry.markerItems) {
        markerItem.style.setProperty("--toc-progress", entry.progress.toFixed(4));
      }
    }
  };

  const setActive = (id) => {
    if (!id || id === activeId) {
      return;
    }

    activeId = id;
    for (const link of tocLinks) {
      const isActive = link.getAttribute("data-toc-target") === id;
      link.classList.toggle("is-active", isActive);
      if (isActive) {
        link.setAttribute("aria-current", "location");
      } else {
        link.removeAttribute("aria-current");
      }
    }
  };

  const activeTargetFromHash = () => {
    if (!window.location.hash) {
      return "";
    }
    try {
      return decodeURIComponent(window.location.hash.slice(1));
    } catch {
      return window.location.hash.slice(1);
    }
  };

  const updateActiveFromScroll = () => {
    updateFrame = undefined;
    if (entries.length === 0) {
      return;
    }

    updateTargetPositions();
    let nextActive = entries[0].id;
    for (const entry of entries) {
      if (window.scrollY + 1 >= entry.scrollTarget) {
        nextActive = entry.id;
      } else {
        break;
      }
    }
    setActive(nextActive);
  };

  const scheduleActiveUpdate = () => {
    if (updateFrame) {
      return;
    }
    updateFrame = window.requestAnimationFrame(updateActiveFromScroll);
  };

  window.addEventListener("scroll", scheduleActiveUpdate, { passive: true });
  window.addEventListener("resize", scheduleActiveUpdate);
  window.addEventListener("hashchange", () => {
    setActive(activeTargetFromHash());
    scheduleActiveUpdate();
  });

  updateTargetPositions();
  setActive(activeTargetFromHash() || (entries[0] && entries[0].id));
  scheduleActiveUpdate();
}
