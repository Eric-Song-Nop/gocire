const sidebarDisclosures = Array.from(document.querySelectorAll("[data-sidebar-disclosure]"));

if (sidebarDisclosures.length > 0) {
  const mobileSidebarQuery = window.matchMedia("(max-width: 720px)");
  const isMobileSidebar = () => mobileSidebarQuery.matches;

  const closeMobileSidebars = () => {
    if (!isMobileSidebar()) {
      return;
    }

    for (const disclosure of sidebarDisclosures) {
      disclosure.removeAttribute("open");
    }
  };

  const syncSidebarDisclosureMode = () => {
    for (const disclosure of sidebarDisclosures) {
      if (isMobileSidebar()) {
        disclosure.removeAttribute("open");
      } else {
        disclosure.setAttribute("open", "");
      }
    }
  };

  for (const disclosure of sidebarDisclosures) {
    const links = Array.from(disclosure.querySelectorAll("a"));
    for (const link of links) {
      link.addEventListener("click", closeMobileSidebars);
    }
  }

  document.addEventListener("click", (event) => {
    if (!isMobileSidebar()) {
      return;
    }

    const target = event.target;
    if (!(target instanceof Element)) {
      return;
    }

    for (const disclosure of sidebarDisclosures) {
      if (disclosure.hasAttribute("open") && !disclosure.contains(target)) {
        disclosure.removeAttribute("open");
      }
    }
  });

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeMobileSidebars();
    }
  });

  if (typeof mobileSidebarQuery.addEventListener === "function") {
    mobileSidebarQuery.addEventListener("change", syncSidebarDisclosureMode);
  } else {
    mobileSidebarQuery.addListener(syncSidebarDisclosureMode);
  }

  syncSidebarDisclosureMode();
}
