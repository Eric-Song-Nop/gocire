const tocLinks = Array.from(document.querySelectorAll("[data-toc-link]"));

if (tocLinks.length > 0) {
  const targetIds = Array.from(new Set(tocLinks.map((link) => link.getAttribute("data-toc-target")).filter(Boolean)));
  const targets = targetIds.map((id) => document.getElementById(id)).filter(Boolean);
  const mobileDisclosures = Array.from(document.querySelectorAll("[data-toc-mobile]"));
  let activeId = "";
  let updateFrame;

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
    if (targets.length === 0) {
      return;
    }

    const offset = Math.min(160, Math.max(72, window.innerHeight * 0.18));
    let nextActive = targets[0].id;
    for (const target of targets) {
      if (target.getBoundingClientRect().top <= offset) {
        nextActive = target.id;
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

  const closeMobileDisclosures = () => {
    for (const disclosure of mobileDisclosures) {
      disclosure.removeAttribute("open");
    }
  };

  for (const disclosure of mobileDisclosures) {
    const links = Array.from(disclosure.querySelectorAll("[data-toc-link]"));
    for (const link of links) {
      link.addEventListener("click", closeMobileDisclosures);
    }
  }

  document.addEventListener("click", (event) => {
    const target = event.target;
    if (!(target instanceof Element)) {
      return;
    }

    for (const disclosure of mobileDisclosures) {
      if (disclosure.hasAttribute("open") && !disclosure.contains(target)) {
        disclosure.removeAttribute("open");
      }
    }
  });

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeMobileDisclosures();
    }
  });

  window.addEventListener("scroll", scheduleActiveUpdate, { passive: true });
  window.addEventListener("resize", scheduleActiveUpdate);
  window.addEventListener("hashchange", () => {
    setActive(activeTargetFromHash());
    scheduleActiveUpdate();
  });

  setActive(activeTargetFromHash() || (targets[0] && targets[0].id));
  scheduleActiveUpdate();
}
