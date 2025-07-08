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
 * @param {string} [fallbackIcon] - Optional fallback icon name.
 * @returns {string}
 */
export function getIcon(item, fallbackIcon = "graph-up") {
  // Support both camelCase and PascalCase keys
  const icon = item.icon || item.Icon;
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
      return `<i class="bi bi-${fallbackIcon} item-icon"></i>`;
  }
}

// --- HTTP Status Text Utility ---

/**
 * Map HTTP status codes to human-readable text.
 * @param {number|string} code
 * @returns {string}
 */
export function getHttpStatusText(code) {
  const statusTexts = {
    100: "Continue",
    101: "Switching Protocols",
    102: "Processing",
    103: "Early Hints",
    200: "OK",
    201: "Created",
    202: "Accepted",
    203: "Non-Authoritative Information",
    204: "No Content",
    205: "Reset Content",
    206: "Partial Content",
    207: "Multi-Status",
    208: "Already Reported",
    226: "IM Used",
    300: "Multiple Choices",
    301: "Moved Permanently",
    302: "Found",
    303: "See Other",
    304: "Not Modified",
    305: "Use Proxy",
    307: "Temporary Redirect",
    308: "Permanent Redirect",
    400: "Bad Request",
    401: "Unauthorized",
    402: "Payment Required",
    403: "Forbidden",
    404: "Not Found",
    405: "Method Not Allowed",
    406: "Not Acceptable",
    407: "Proxy Authentication Required",
    408: "Request Timeout",
    409: "Conflict",
    410: "Gone",
    411: "Length Required",
    412: "Precondition Failed",
    413: "Payload Too Large",
    414: "URI Too Long",
    415: "Unsupported Media Type",
    416: "Range Not Satisfiable",
    417: "Expectation Failed",
    418: "I'm a teapot",
    421: "Misdirected Request",
    422: "Unprocessable Entity",
    423: "Locked",
    424: "Failed Dependency",
    425: "Too Early",
    426: "Upgrade Required",
    428: "Precondition Required",
    429: "Too Many Requests",
    431: "Request Header Fields Too Large",
    451: "Unavailable For Legal Reasons",
    500: "Internal Server Error",
    501: "Not Implemented",
    502: "Bad Gateway",
    503: "Service Unavailable",
    504: "Gateway Timeout",
    505: "HTTP Version Not Supported",
    506: "Variant Also Negotiates",
    507: "Insufficient Storage",
    508: "Loop Detected",
    510: "Not Extended",
    511: "Network Authentication Required",
  };
  return statusTexts[code] || "Unknown Status";
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
