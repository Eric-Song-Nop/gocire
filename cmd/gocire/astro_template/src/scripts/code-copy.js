const generatedCodeBlocks = Array.from(document.querySelectorAll("[data-code-block]"));
const proseCodeSurfaces = Array.from(document.querySelectorAll(".page-content .cire-prose > pre, .page-content .cire-prose > .chroma"));
const codeBlocks = [...generatedCodeBlocks, ...proseCodeSurfaces.map((surface) => wrapProseCodeSurface(surface))].filter(Boolean);
const resetDelayMs = 1600;
const idleLabel = "Copy code";
const copiedLabel = "Copied";
const failedLabel = "Copy failed";
const copyIconTemplate = document.getElementById("gocire-code-copy-icon");
const copiedIconTemplate = document.getElementById("gocire-code-copy-success-icon");

if (codeBlocks.length > 0) {
  for (const block of codeBlocks) {
    enhanceCodeBlock(block);
  }
}

function wrapProseCodeSurface(surface) {
  if (!(surface instanceof HTMLElement) || surface.closest("[data-code-block]")) {
    return null;
  }

  const wrapper = document.createElement("div");
  wrapper.className = "cire-code-block cire-code-block--prose";
  wrapper.setAttribute("data-code-block", "");
  surface.before(wrapper);
  wrapper.appendChild(surface);
  return wrapper;
}

function enhanceCodeBlock(block) {
  if (!(block instanceof HTMLElement) || block.querySelector("[data-code-copy]")) {
    return;
  }

  const code = codeElementForBlock(block);
  if (!code) {
    return;
  }

  let resetTimer;
  const button = document.createElement("button");
  button.type = "button";
  button.className = "cire-code-copy";
  button.setAttribute("data-code-copy", "");
  setButtonState(button, "idle");

  button.addEventListener("click", async () => {
    window.clearTimeout(resetTimer);
    button.disabled = true;

    try {
      await writeClipboard(copyTextForCode(code));
      setButtonState(button, "copied");
      resetTimer = window.setTimeout(() => setButtonState(button, "idle"), resetDelayMs);
    } catch {
      setButtonState(button, "failed");
      resetTimer = window.setTimeout(() => setButtonState(button, "idle"), resetDelayMs);
    } finally {
      button.disabled = false;
    }
  });

  block.appendChild(button);
}

function codeElementForBlock(block) {
  return block.querySelector("code") || block.querySelector("pre");
}

function copyTextForCode(code) {
  const clone = code.cloneNode(true);
  if (!(clone instanceof Element)) {
    return "";
  }

  for (const hint of clone.querySelectorAll("[data-inlay-hint]")) {
    hint.remove();
  }
  return clone.textContent || "";
}

async function writeClipboard(text) {
  if (navigator.clipboard && window.isSecureContext) {
    await navigator.clipboard.writeText(text);
    return;
  }
  writeClipboardFallback(text);
}

function writeClipboardFallback(text) {
  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.setAttribute("readonly", "");
  textarea.style.position = "fixed";
  textarea.style.left = "-9999px";
  textarea.style.top = "0";
  document.body.appendChild(textarea);
  textarea.select();

  try {
    if (!document.execCommand("copy")) {
      throw new Error("copy command failed");
    }
  } finally {
    textarea.remove();
  }
}

function setButtonState(button, state) {
  const label = labelForState(state);
  button.dataset.copyState = state;
  button.setAttribute("aria-label", label);
  button.setAttribute("title", label);
  button.replaceChildren();

  const icon = iconForState(state);
  if (icon) {
    button.appendChild(icon);
    return;
  }
  button.textContent = state === "copied" ? copiedLabel : "Copy";
}

function labelForState(state) {
  if (state === "copied") {
    return copiedLabel;
  }
  if (state === "failed") {
    return failedLabel;
  }
  return idleLabel;
}

function iconForState(state) {
  const template = state === "copied" ? copiedIconTemplate : copyIconTemplate;
  if (!(template instanceof HTMLTemplateElement)) {
    return null;
  }
  return template.content.cloneNode(true);
}
