/**
 * lmon mobile.js
 * Refactored to use shared utilities from utils.js for maintainability and DRY code.
 */

import {
  getStatusClass,
  getIcon,
  normalizeItems,
  fetchJson,
  handleFetchError,
  createCountdown,
} from "./utils.js";

// Utility: get type label for mobile view
function getTypeLabel(type) {
  switch (type) {
    case "cpu":
    case "memory":
    case "system":
      return "System";
    case "disk":
      return "Disk";
    case "health":
      return "Health";
    default:
      return type.charAt(0).toUpperCase() + type.slice(1);
  }
}

// Utility: sort items by status for mobile view
function sortItemsByStatus(items) {
  const statusOrder = {
    error: 0,
    critical: 0,
    warning: 1,
    ok: 2,
    unknown: 3,
  };
  return items.slice().sort((a, b) => {
    const aOrder =
      statusOrder[a.status] !== undefined ? statusOrder[a.status] : 99;
    const bOrder =
      statusOrder[b.status] !== undefined ? statusOrder[b.status] : 99;
    return aOrder - bOrder;
  });
}

// Render all items as a single sorted list for mobile
function renderMobileItems(items) {
  const sorted = sortItemsByStatus(items);
  if (sorted.length === 0) {
    return '<div class="text-center">No items to display</div>';
  }
  let html = "";
  sorted.forEach((item, idx) => {
    const statusClass = getStatusClass(item.status);
    const icon = getIcon(item);
    const typeLabel = getTypeLabel(item.type);
    const bgClass = idx % 2 === 0 ? "even" : "odd";
    const detail = item.value;
    const detail2 = item.value2;
    html += `
      <div class="mobile-list-item ${bgClass}" data-id="${item.id}">
        <div class="mobile-line1">
          <div>${icon}</div>
          <div class="status-indicator ${statusClass}"></div>
          <div>${item.name}</div>
          <div class="type-label ms-auto">${typeLabel}</div>
        </div>
        <div class="mobile-line2">
          <span class="mobile-status-badge">
            <span class="badge ${statusClass}">${item.status}</span>
          </span>
          <span class="mobile-detail">${detail}
        </div>
        <div class="mobile-line3">
          <span class="mobile-detail2">${detail2}</span>
        </div>
      </div>
    `;
  });
  return html;
}

// Load and render mobile items
async function loadMobileItems() {
  try {
    const itemsMap = await fetchJson("/api/items");
    const items = normalizeItems(itemsMap);
    document.getElementById("mobile-items-list").innerHTML =
      renderMobileItems(items);
  } catch (error) {
    handleFetchError(error, "Error loading items");
    document.getElementById("mobile-items-list").innerHTML =
      '<div class="alert alert-danger">Error loading items</div>';
  }
}

// --- Countdown/Refresh Logic ---
let countdown;
let mobileRefreshInterval = 60000; // fallback default

function fetchMobileIntervalAndStart() {
  fetchJson("/api/config")
    .then((config) => {
      if (config && config.Monitoring && config.Monitoring.Interval) {
        mobileRefreshInterval = config.Monitoring.Interval * 1000;
      } else {
        mobileRefreshInterval = 60000;
      }
      setupCountdownAndStart();
    })
    .catch(() => {
      mobileRefreshInterval = 60000;
      setupCountdownAndStart();
    });
}

function setupCountdownAndStart() {
  const countdownElement = document.getElementById("refresh-countdown");
  countdown = createCountdown(
    countdownElement,
    loadMobileItems,
    mobileRefreshInterval,
  );
  loadMobileItems();
  countdown.start();
}

// Mobile detection and toggle logic
function isMobile() {
  return window.innerWidth <= 575;
}

function showMobileToggle() {
  const desktopLink = document.getElementById("toggle-desktop-link");
  const desktopBtn = document.getElementById("toggle-desktop-btn");
  if (isMobile()) {
    if (desktopLink) desktopLink.style.display = "";
    if (desktopBtn) desktopBtn.style.display = "";
  } else {
    if (desktopLink) desktopLink.style.display = "none";
    if (desktopBtn) desktopBtn.style.display = "none";
  }
}

document.addEventListener("DOMContentLoaded", function () {
  fetchMobileIntervalAndStart();

  // Attach refresh button handler
  const refreshBtn = document.getElementById("refresh-btn");
  if (refreshBtn) {
    refreshBtn.addEventListener("click", function () {
      loadMobileItems();
      if (countdown) countdown.reset();
    });
  }

  showMobileToggle();
  window.addEventListener("resize", showMobileToggle);

  // Offer to redirect to mobile if on mobile and not already here
  if (isMobile() && window.location.pathname !== "/mobile") {
    if (
      confirm("You appear to be on a mobile device. Switch to mobile view?")
    ) {
      window.location.href = "/mobile";
    }
  }
});
