/**
 * lmon utils.js
 * Shared utility functions for dashboard, mobile, and config pages.
 * Includes: status/icon rendering, HTTP status text, toast, item normalization, and fetch helpers.
 */

// --- Status and Icon Utilities ---

/**
 * Get the CSS class for a given status string.
 * @param {string} status
 * @returns {string}
 */
export function getStatusClass(status) {
  if (!status) return "status-unknown";
  switch (status.toLowerCase()) {
    case "ok":
      return "status-ok";
    case "warning":
      return "status-warning";
    case "critical":
      return "status-critical";
    case "error":
      return "status-critical";
    default:
      return "status-unknown";
  }
}

/**
 * Get the HTML for an icon based on item type or icon property.
 * @param {object} item
 * @returns {string}
 */
export function getIcon(item) {
  // Support both camelCase and PascalCase keys
  const icon = item.icon;
  if (icon) {
    return `<i class="bi bi-${icon} item-icon"></i>`;
  }
  switch (item.type) {
    case "cpu":
      return '<i class="bi bi-cpu item-icon"></i>';
    case "memory":
      return '<i class="bi bi-speedometer item-icon"></i>';
    case "disk":
      return '<i class="bi bi-hdd item-icon"></i>';
    case "health":
      return '<i class="bi bi-activity item-icon"></i>';
    default:
      return `<i class="bi bi-folder item-icon"></i>`;
  }
}

// --- Toast/Notification Utility ---

/**
 * Show a toast notification (Bootstrap 5).
 * @param {string} title
 * @param {string} message
 * @param {boolean} [isError=false]
 */
export function showToast(title, message, isError = false) {
  const toastEl = document.getElementById("toast");
  const toastTitle = document.getElementById("toast-title");
  const toastMessage = document.getElementById("toast-message");
  if (!toastEl || !toastTitle || !toastMessage) return;

  toastTitle.textContent = title;
  toastMessage.textContent = message;

  if (isError) {
    toastEl.classList.add("bg-danger", "text-white");
  } else {
    toastEl.classList.remove("bg-danger", "text-white");
  }

  const toast = new bootstrap.Toast(toastEl);
  toast.show();
}

// --- Item Normalization Utility ---

/**
 * Normalize an API item map to an array of item objects with consistent fields.
 * @param {object} itemsMap - API response object: { id: Result, ... }
 * @returns {Array<object>}
 */
export function normalizeItems(itemsMap) {
  return Object.entries(itemsMap).map(([id, result]) => {
    // Try to infer type from group or id
    let type = "";
    if (result.Group === "system") {
      if (id.endsWith("cpu")) type = "cpu";
      else if (id.endsWith("mem")) type = "memory";
      else type = "system";
    } else if (result.Group === "disk") {
      type = "disk";
    } else if (result.Group === "health") {
      type = "health";
    } else {
      type = result.Group || "";
    }
    // Map numeric status codes to string values
    const statusMap = {
      0: "unknown",
      1: "error",
      2: "critical",
      3: "warning",
      4: "ok",
    };
    let status = "unknown";
    if (typeof result.Status === "number") {
      status = statusMap[result.Status] || "unknown";
    } else if (typeof result.Status === "string") {
      status = result.Status.toLowerCase();
    }
    return {
      id,
      type,
      name: result.DisplayName || id,
      status,
      value: result.Value || "",
      value2: result.Value2 || "",
      unit: result.Unit || "",
      threshold: result.Threshold || null,
      last_check: result.LastCheck || "",
      message: result.Message || "",
      icon: result.Icon || result.icon || "",
    };
  });
}

// --- Fetch Helpers ---

/**
 * Wrapper for fetch with JSON parsing and error handling.
 * @param {string} url
 * @param {object} [options]
 * @returns {Promise<any>}
 */
export async function fetchJson(url, options = {}) {
  const response = await fetch(url, options);
  if (!response.ok) {
    let msg = `HTTP ${response.status}`;
    try {
      const data = await response.json();
      msg = data.message || msg;
    } catch {
      // fallback to status text
      msg = response.statusText || msg;
    }
    throw new Error(msg);
  }
  const contentType = response.headers.get("content-type") || "";
  if (contentType.includes("application/json")) {
    return response.json();
  } else {
    return response.text();
  }
}

/**
 * Standard error handler for fetch calls.
 * @param {Error} error
 * @param {string} [context]
 */
export function handleFetchError(error, context = "Request failed") {
  console.error(context, error);
  showToast("Error", error.message || context, true);
}

// --- Countdown/Refresh Helpers ---

/**
 * Countdown timer utility for refresh buttons.
 * @param {HTMLElement} countdownElement
 * @param {() => void} onExpire
 * @param {number} intervalMs
 * @returns {object} { reset, stop }
 */
export function createCountdown(countdownElement, onExpire, intervalMs) {
  let nextRefreshTime = 0;
  let intervalId = null;
  let countdownId = null;

  function updateCountdown() {
    const now = Date.now();
    const timeLeft = Math.max(0, nextRefreshTime - now);
    const secondsLeft = Math.ceil(timeLeft / 1000);
    if (countdownElement) {
      countdownElement.textContent = secondsLeft > 0 ? `(${secondsLeft}s)` : "";
    }
  }

  function reset() {
    nextRefreshTime = Date.now() + intervalMs;
    updateCountdown();
  }

  function start() {
    reset();
    if (intervalId) clearInterval(intervalId);
    if (countdownId) clearInterval(countdownId);
    intervalId = setInterval(() => {
      onExpire();
      reset();
    }, intervalMs);
    countdownId = setInterval(updateCountdown, 1000);
  }

  function stop() {
    if (intervalId) clearInterval(intervalId);
    if (countdownId) clearInterval(countdownId);
  }

  return { start, reset, stop };
}
