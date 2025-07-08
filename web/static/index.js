/**
 * lmon index.js
 * Dashboard logic extracted from index.html for maintainability and caching.
 */

// Function to get status class
function getStatusClass(status) {
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

// Function to get icon
function getIcon(item) {
  if (item.icon) {
    return `<i class="bi bi-${item.icon} item-icon"></i>`;
  }

  // Default icons based on type
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
      return '<i class="bi bi-graph-up item-icon"></i>';
  }
}

// Function to get HTTP status text
function getHttpStatusText(code) {
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

// Function to load items
function loadItems() {
  fetch("/api/items")
    .then((response) => response.json())
    .then((itemsMap) => {
      // itemsMap is an object: { id: Result, ... }
      // We need to convert it to an array and infer type/group from keys or values
      const items = Object.entries(itemsMap).map(([id, result]) => {
        // Try to infer type from group or id
        let type = "";
        if (result.Group === "system") {
          if (id.endsWith("cpu")) type = "cpu";
          else if (id.endsWith("mem")) type = "memory";
          else type = "system";
        } else if (result.Group === "filesystem") {
          type = "disk";
        } else if (result.Group === "healthcheck" || result.Group === "app") {
          type = "health";
        } else {
          type = result.Group || "";
        }
        // Try to extract threshold, unit, last_check, message if present (may not be)
        return {
          id,
          type,
          name: result.DisplayName || id,
          // Map numeric status codes to string values
          status: (function () {
            const statusMap = {
              0: "unknown",
              1: "error",
              2: "critical",
              3: "warning",
              4: "ok",
            };
            if (typeof result.Status === "number") {
              return statusMap[result.Status] || "unknown";
            }
            if (typeof result.Status === "string") {
              return result.Status.toLowerCase();
            }
            return "unknown";
          })(),
          value: result.Value || "",
          unit: result.Unit || "", // May not exist
          threshold: result.Threshold || null, // May not exist
          last_check: result.LastCheck || "", // May not exist
          message: result.Message || "", // May not exist
          icon: result.Icon || "", // May not exist
        };
      });

      const systemItems = items
        .filter((item) => item.type === "cpu" || item.type === "memory")
        .sort((a, b) => a.name.localeCompare(b.name));
      const diskItems = items.filter((item) => item.type === "disk");
      const healthItems = items.filter((item) => item.type === "health");

      // Render system items
      document.getElementById("system-items").innerHTML =
        renderItems(systemItems);

      // Render disk items
      document.getElementById("disk-items").innerHTML = renderItems(diskItems);

      // Render health items
      document.getElementById("health-items").innerHTML =
        renderItems(healthItems);

      // Add click handlers for item details
      document.querySelectorAll(".item-row").forEach((row) => {
        row.addEventListener("click", function () {
          const itemId = this.getAttribute("data-id");
          showItemDetails(itemId, items);
        });
      });
    })
    .catch((error) => {
      console.error("Error loading items:", error);
      document.getElementById("system-items").innerHTML =
        '<div class="alert alert-danger">Error loading system items</div>';
      document.getElementById("disk-items").innerHTML =
        '<div class="alert alert-danger">Error loading disk items</div>';
      document.getElementById("health-items").innerHTML =
        '<div class="alert alert-danger">Error loading health items</div>';
    });
}

// Function to render items
function renderItems(items) {
  if (items.length === 0) {
    return '<div class="text-center">No items to display</div>';
  }

  let html = '<div class="list-group">';

  items.forEach((item) => {
    const statusClass = getStatusClass(item.status);
    const icon = getIcon(item);

    // Determine if item should be expanded (if status is not OK)
    const expanded = item.status.toLowerCase() !== "ok" ? "show" : "";

    html += `
            <div class="list-group-item item-row" data-id="${item.id}">
                <div class="d-flex justify-content-between align-items-center">
                    <div>
                        <span style="display: inline-block;">${icon}</span>
                        <span class="status-indicator ${statusClass}"></span>
                        ${item.name}
                    </div>
                    <div>
                        <span class="badge ${statusClass}">${item.status}</span>
                        <span class="ms-2">${
                          item.type === "health"
                            ? typeof item.value === "number"
                              ? `(${item.value.toFixed(0)}) ${getHttpStatusText(item.value)}`
                              : item.value
                            : `${item.unit === "%" ? parseFloat(item.value).toFixed(2) : item.value}${item.unit}`
                        }</span>
                    </div>
                </div>
            </div>
        `;
  });

  html += "</div>";
  return html;
}

// Function to show item details
function showItemDetails(itemId, items) {
  const item = items.find((i) => i.id === itemId);
  if (!item) return;

  const statusClass = getStatusClass(item.status);
  const icon = getIcon(item);

  let html = `
        <div class="d-flex align-items-center mb-3">
            ${icon}
            <h4 class="mb-0">${item.name}</h4>
        </div>
        <div class="mb-3">
            <span class="badge ${statusClass} fs-6">${item.status}</span>
        </div>
        <div class="mb-3">
            <strong>Value:</strong> ${
              item.type === "health"
                ? typeof item.value === "number"
                  ? `(${item.value.toFixed(0)}) ${getHttpStatusText(item.value)}`
                  : item.value
                : (item.unit === "%" && !isNaN(parseFloat(item.value))
                    ? parseFloat(item.value).toFixed(2)
                    : item.value) + (item.unit || "")
            }
        </div>
        <div class="mb-3">
            <strong>Threshold:</strong> ${
              item.threshold !== undefined &&
              item.threshold !== null &&
              item.threshold !== 0
                ? item.unit === "%"
                  ? parseFloat(item.threshold).toFixed(2)
                  : item.threshold
                : "N/A"
            }${item.threshold !== undefined && item.threshold !== null && item.threshold !== 0 ? item.unit : ""}
        </div>
        <div class="mb-3">
            <strong>Last Check:</strong> ${item.last_check ? new Date(item.last_check).toLocaleString() : "N/A"}
        </div>
        <div class="mb-3">
            <strong>Message:</strong> ${item.message || ""}
        </div>
    `;

  document.getElementById("modal-title").textContent = item.name;
  document.getElementById("modal-body").innerHTML = html;

  const modal = new bootstrap.Modal(document.getElementById("itemDetailModal"));
  modal.show();
}

// Variables for refresh countdown
let nextRefreshTime = 0;
let refreshInterval = 60000; // Default to 60 seconds (will be updated from server config)

// Function to update countdown display
function updateCountdown() {
  const now = Date.now();
  const timeLeft = Math.max(0, nextRefreshTime - now);
  const secondsLeft = Math.ceil(timeLeft / 1000);

  const countdownElement = document.getElementById("refresh-countdown");
  if (secondsLeft > 0) {
    countdownElement.textContent = `(${secondsLeft}s)`;
  } else {
    countdownElement.textContent = "";
  }
}

// Function to reset countdown timer
function resetCountdown() {
  nextRefreshTime = Date.now() + refreshInterval;
  updateCountdown();
}

// Enhanced loadItems function that resets the countdown
function loadItemsWithCountdown() {
  loadItems();
  resetCountdown();
}

// Load items on page load
document.addEventListener("DOMContentLoaded", function () {
  // Get refresh interval from data attribute (in seconds) and convert to milliseconds
  const refreshBtn = document.getElementById("refresh-btn");
  const refreshIntervalSeconds =
    parseInt(refreshBtn.getAttribute("data-refresh-interval")) || 60;
  refreshInterval = refreshIntervalSeconds * 1000;

  loadItems();
  resetCountdown();

  // Set up refresh button
  refreshBtn.addEventListener("click", loadItemsWithCountdown);

  // Auto-refresh using the configured interval
  setInterval(loadItemsWithCountdown, refreshInterval);

  // Update countdown every second
  setInterval(updateCountdown, 1000);
});
