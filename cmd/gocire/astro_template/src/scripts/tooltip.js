import { autoUpdate, computePosition, flip, offset, shift } from "@floating-ui/dom";

const hoverSelector = "[data-hover-html], [data-hover]";
const tokens = Array.from(document.querySelectorAll(hoverSelector));

if (tokens.length > 0) {
  const hideDelayMs = 120;
  const tapMoveThreshold = 8;
  const focusableSelector = [
    "a[href]",
    "button:not([disabled])",
    "input:not([disabled])",
    "select:not([disabled])",
    "textarea:not([disabled])",
    "[tabindex]:not([tabindex=\"-1\"])",
  ].join(", ");
  let tooltip;
  let cleanupPosition;
  let activeToken;
  let hideTimer;
  let mode = "closed";
  let pointerCandidate;
  let suppressClickToken;
  let suppressClickTimer;

  const ensureTooltip = () => {
    if (tooltip) {
      return tooltip;
    }

    tooltip = document.createElement("div");
    tooltip.id = "gocire-tooltip";
    tooltip.className = "gocire-tooltip";
    tooltip.setAttribute("role", "dialog");
    tooltip.setAttribute("aria-label", "Symbol information");
    tooltip.setAttribute("aria-modal", "false");
    tooltip.setAttribute("tabindex", "-1");
    tooltip.hidden = true;

    const content = document.createElement("div");
    content.className = "gocire-tooltip__content";
    tooltip.appendChild(content);

    const actions = document.createElement("div");
    actions.className = "gocire-tooltip__actions";
    actions.hidden = true;
    const actionLink = document.createElement("a");
    actionLink.className = "gocire-tooltip__action";
    actionLink.textContent = "Open link";
    actions.appendChild(actionLink);
    tooltip.appendChild(actions);

    document.body.appendChild(tooltip);
    tooltip.addEventListener("mouseenter", cancelHide);
    tooltip.addEventListener("mouseleave", scheduleHide);
    tooltip.addEventListener("focusin", cancelHide);
    tooltip.addEventListener("focusout", scheduleHide);
    return tooltip;
  };

  const decodeBase64 = (encoded) => {
    if (!encoded) {
      return "";
    }

    try {
      return new TextDecoder().decode(Uint8Array.from(atob(encoded), (char) => char.charCodeAt(0)));
    } catch {
      return "";
    }
  };

  const closestToken = (target) => {
    return target instanceof Element ? target.closest(hoverSelector) : null;
  };

  const isTouchPointer = (event) => {
    return event.pointerType === "touch" || event.pointerType === "pen";
  };

  const tokenHref = (token) => {
    return token instanceof HTMLAnchorElement ? token.getAttribute("href") : "";
  };

  const isFocusableElement = (element) => {
    return element instanceof HTMLElement && !element.hidden && element.getAttribute("aria-hidden") !== "true";
  };

  const tooltipFocusableElements = () => {
    if (!tooltip || tooltip.hidden) {
      return [];
    }

    return Array.from(tooltip.querySelectorAll(focusableSelector)).filter(isFocusableElement);
  };

  const pageFocusableElements = () => {
    return Array.from(document.querySelectorAll(focusableSelector)).filter((element) => {
      return isFocusableElement(element) && (!tooltip || !tooltip.contains(element));
    });
  };

  const focusElement = (element) => {
    if (!(element instanceof HTMLElement)) {
      return false;
    }

    try {
      element.focus({ preventScroll: true });
    } catch {
      element.focus();
    }
    return document.activeElement === element;
  };

  const focusFirstTooltipItem = () => {
    const focusable = tooltipFocusableElements();
    if (focusable.length === 0) {
      return false;
    }

    cancelHide();
    return focusElement(focusable[0]);
  };

  const focusPageElementAdjacentToToken = (token, direction) => {
    if (!(token instanceof HTMLElement)) {
      return false;
    }

    const focusable = pageFocusableElements();
    const tokenIndex = focusable.indexOf(token);
    if (tokenIndex === -1) {
      return false;
    }

    return focusElement(focusable[tokenIndex + direction]);
  };

  const setTooltipAction = (floating, token) => {
    const actions = floating.querySelector(".gocire-tooltip__actions");
    const actionLink = floating.querySelector(".gocire-tooltip__action");
    if (!actions || !(actionLink instanceof HTMLAnchorElement)) {
      return;
    }

    const href = tokenHref(token);
    if (!href) {
      actions.hidden = true;
      actionLink.removeAttribute("href");
      return;
    }

    actionLink.href = href;
    actionLink.textContent = "Open link";
    actions.hidden = false;
  };

  const setTooltipContent = (floating, token) => {
    const content = floating.querySelector(".gocire-tooltip__content");
    if (!content) {
      return false;
    }

    const html = decodeBase64(token.getAttribute("data-hover-html"));
    if (html) {
      content.innerHTML = html;
      setTooltipAction(floating, token);
      return true;
    }

    const text = decodeBase64(token.getAttribute("data-hover"));
    if (text) {
      content.textContent = text;
      setTooltipAction(floating, token);
      return true;
    }

    content.textContent = "";
    setTooltipAction(floating, token);
    return false;
  };

  const updatePosition = async (token) => {
    const floating = ensureTooltip();
    const { x, y } = await computePosition(token, floating, {
      placement: "top-start",
      strategy: "fixed",
      middleware: [offset(8), flip(), shift({ padding: 12 })],
    });

    Object.assign(floating.style, {
      left: x + "px",
      top: y + "px",
    });
  };

  const stopPositionUpdates = () => {
    if (cleanupPosition) {
      cleanupPosition();
      cleanupPosition = undefined;
    }
  };

  const cancelHide = () => {
    if (hideTimer) {
      window.clearTimeout(hideTimer);
      hideTimer = undefined;
    }
  };

  const clearTokenState = (token) => {
    if (!token) {
      return;
    }

    token.removeAttribute("aria-describedby");
    token.removeAttribute("aria-controls");
    token.removeAttribute("aria-expanded");
  };

  const isTooltipActive = () => {
    if (!activeToken || !tooltip || tooltip.hidden) {
      return false;
    }
    if (mode === "touchPinned" || mode === "keyboardPinned") {
      return true;
    }

    const activeElement = document.activeElement;
    if (activeElement instanceof Node && (activeToken.contains(activeElement) || tooltip.contains(activeElement))) {
      return true;
    }

    return activeToken.matches(":hover") || tooltip.matches(":hover");
  };

  const hideTooltip = (token, options = {}) => {
    if (token && activeToken && token !== activeToken) {
      return;
    }
    if (!options.force && isTooltipActive()) {
      return;
    }

    cancelHide();
    stopPositionUpdates();
    clearTokenState(activeToken);
    activeToken = undefined;
    pointerCandidate = undefined;
    mode = "closed";

    if (tooltip) {
      tooltip.hidden = true;
    }
  };

  const scheduleHide = () => {
    cancelHide();
    hideTimer = window.setTimeout(() => {
      hideTooltip(activeToken);
    }, hideDelayMs);
  };

  const showTooltip = (token, nextMode = "hover") => {
    const floating = ensureTooltip();
    if (!setTooltipContent(floating, token)) {
      hideTooltip(undefined, { force: true });
      return;
    }

    cancelHide();
    floating.hidden = false;
    if (activeToken && activeToken !== token) {
      clearTokenState(activeToken);
    }
    activeToken = token;
    mode = nextMode;
    token.setAttribute("aria-describedby", floating.id);
    token.setAttribute("aria-controls", floating.id);
    token.setAttribute("aria-expanded", "true");

    stopPositionUpdates();
    cleanupPosition = autoUpdate(token, floating, () => {
      updatePosition(token);
    });
    updatePosition(token);
  };

  const suppressNextClick = (token) => {
    suppressClickToken = token;
    if (suppressClickTimer) {
      window.clearTimeout(suppressClickTimer);
    }
    suppressClickTimer = window.setTimeout(() => {
      if (suppressClickToken === token) {
        suppressClickToken = undefined;
      }
      suppressClickTimer = undefined;
    }, 800);
  };

  const handleTouchTap = (token) => {
    suppressNextClick(token);
    if (activeToken === token && mode === "touchPinned") {
      hideTooltip(token, { force: true });
      return;
    }

    showTooltip(token, "touchPinned");
  };

  const handleKeyboardActivation = (token, event) => {
    if (token instanceof HTMLAnchorElement || (event.key !== "Enter" && event.key !== " ")) {
      return;
    }

    event.preventDefault();
    event.stopPropagation();
    if (activeToken === token && mode === "keyboardPinned") {
      hideTooltip(token, { force: true });
      return;
    }

    showTooltip(token, "keyboardPinned");
    focusFirstTooltipItem();
  };

  const handleTooltipTab = (event) => {
    if (event.key !== "Tab" || !tooltip || tooltip.hidden || !activeToken) {
      return;
    }

    const focusable = tooltipFocusableElements();
    if (focusable.length === 0) {
      return;
    }

    const activeElement = document.activeElement;
    if (activeElement === activeToken && !event.shiftKey) {
      event.preventDefault();
      focusFirstTooltipItem();
      return;
    }

    if (!(activeElement instanceof Node) || !tooltip.contains(activeElement)) {
      return;
    }

    const focusIndex = focusable.indexOf(activeElement);
    if (event.shiftKey && focusIndex === 0) {
      event.preventDefault();
      focusElement(activeToken);
      return;
    }

    if (!event.shiftKey && focusIndex === focusable.length - 1) {
      const tokenToLeave = activeToken;
      event.preventDefault();
      hideTooltip(tokenToLeave, { force: true });
      focusPageElementAdjacentToToken(tokenToLeave, 1);
    }
  };

  for (const token of tokens) {
    if (!token.hasAttribute("tabindex")) {
      token.setAttribute("tabindex", "0");
    }
    if (!(token instanceof HTMLAnchorElement) && !token.hasAttribute("role")) {
      token.setAttribute("role", "button");
    }

    token.addEventListener("mouseenter", () => {
      if (mode !== "touchPinned") {
        showTooltip(token, "hover");
      }
    });
    token.addEventListener("mouseleave", scheduleHide);
    token.addEventListener("focus", () => {
      if (mode !== "touchPinned" || activeToken !== token) {
        showTooltip(token, "focus");
      }
    });
    token.addEventListener("blur", scheduleHide);
    token.addEventListener("keydown", (event) => handleKeyboardActivation(token, event));
  }

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      const shouldRestoreFocus = tooltip && document.activeElement instanceof Node && tooltip.contains(document.activeElement);
      const tokenToRestore = activeToken;
      hideTooltip(activeToken, { force: true });
      if (shouldRestoreFocus && tokenToRestore instanceof HTMLElement) {
        tokenToRestore.focus({ preventScroll: true });
      }
      return;
    }
    handleTooltipTab(event);
  });

  document.addEventListener(
    "pointerdown",
    (event) => {
      const target = event.target instanceof Node ? event.target : null;
      const hoveredToken = closestToken(event.target);

      if (tooltip && target && tooltip.contains(target)) {
        cancelHide();
        return;
      }

      if (isTouchPointer(event) && hoveredToken) {
        pointerCandidate = {
          pointerId: event.pointerId,
          token: hoveredToken,
          x: event.clientX,
          y: event.clientY,
        };
        return;
      }

      if (!hoveredToken && (!tooltip || !target || !tooltip.contains(target))) {
        hideTooltip(activeToken, { force: true });
      }
    },
    true,
  );

  document.addEventListener(
    "pointermove",
    (event) => {
      if (!pointerCandidate || pointerCandidate.pointerId !== event.pointerId) {
        return;
      }

      const dx = event.clientX - pointerCandidate.x;
      const dy = event.clientY - pointerCandidate.y;
      if (Math.hypot(dx, dy) > tapMoveThreshold) {
        pointerCandidate = undefined;
      }
    },
    true,
  );

  document.addEventListener(
    "pointerup",
    (event) => {
      if (!pointerCandidate || pointerCandidate.pointerId !== event.pointerId) {
        return;
      }

      const token = pointerCandidate.token;
      pointerCandidate = undefined;
      handleTouchTap(token);
      event.preventDefault();
      event.stopPropagation();
    },
    true,
  );

  document.addEventListener(
    "pointercancel",
    (event) => {
      if (pointerCandidate && pointerCandidate.pointerId === event.pointerId) {
        pointerCandidate = undefined;
      }
    },
    true,
  );

  document.addEventListener(
    "click",
    (event) => {
      const clickedToken = closestToken(event.target);
      if (clickedToken && suppressClickToken === clickedToken) {
        suppressClickToken = undefined;
        event.preventDefault();
        event.stopPropagation();
      }
    },
    true,
  );
}
