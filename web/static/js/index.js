/**
 * lmon index.js
 * Dashboard logic refactored to use shared utilities from utils.js.
 */

import {
  getStatusClass,
  getIcon,
  normalizeItems,
  fetchJson,
  handleFetchError,
  createCountdown,
} from "./utils.js";

// Function to render items
function renderItems(items) {
  if (items.length === 0) {
    return '<div class="text-center">No items to display</div>';
  }

  let html = '<div class="list-group">';
  items.forEach((item) => {
    const statusClass = getStatusClass(item.status);
    const icon = getIcon(item);

    html += `
            <div class="list-group-item item-row item-container" data-id="${item.id}">
                <div class="d-flex justify-content-between align-items-start">
                    <div class="item-status">
                        <span style="display: inline-block;">${icon}</span>
                        <span class="status-indicator ${statusClass}"></span>
                    </div>
                    <div class="item-stack">
                      <div class="item-detail1">
                          <span>${item.name}</span>
                          <span class="badge ${statusClass}">${item.status}</span>
                          <span class="ms-2">${item.value}</span>
                      </div>
                      <div class="item-detail2">
                          <span class="ms-2">${item.value2}</span>
                      </div>
                    </div>
                </div>
            </div>
        `;
  });

  html += "</div>";
  return html;
}

// Function to show item details in modal
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
            <strong>Value:</strong> ${item.value}
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

// Function to load and render items
async function loadItems() {
  try {
    const itemsMap = await fetchJson("/api/items");
    const items = normalizeItems(itemsMap);

    const systemItems = items
      .filter((item) => item.type === "cpu" || item.type === "memory")
      .sort((a, b) => a.name.localeCompare(b.name));
    const diskItems = items.filter((item) => item.type === "disk");
    const healthItems = items.filter((item) => item.type === "health");

    document.getElementById("system-items").innerHTML =
      renderItems(systemItems);
    document.getElementById("disk-items").innerHTML = renderItems(diskItems);
    document.getElementById("health-items").innerHTML =
      renderItems(healthItems);

    // Add click handlers for item details
    document.querySelectorAll(".item-row").forEach((row) => {
      row.addEventListener("click", function () {
        const itemId = this.getAttribute("data-id");
        showItemDetails(itemId, items);
      });
    });
  } catch (error) {
    handleFetchError(error, "Error loading items");
    document.getElementById("system-items").innerHTML =
      '<div class="alert alert-danger">Error loading system items</div>';
    document.getElementById("disk-items").innerHTML =
      '<div class="alert alert-danger">Error loading disk items</div>';
    document.getElementById("health-items").innerHTML =
      '<div class="alert alert-danger">Error loading health items</div>';
  }
}

// --- Countdown/Refresh Logic ---
let countdown;
document.addEventListener("DOMContentLoaded", function () {
  const refreshBtn = document.getElementById("refresh-btn");
  const countdownElement = document.getElementById("refresh-countdown");
  const refreshIntervalSeconds =
    parseInt(refreshBtn.getAttribute("data-refresh-interval")) || 60;
  const refreshInterval = refreshIntervalSeconds * 1000;

  // Setup countdown utility
  countdown = createCountdown(countdownElement, loadItems, refreshInterval);

  // Initial load and start countdown
  loadItems();
  countdown.start();

  // Set up refresh button
  refreshBtn.addEventListener("click", () => {
    loadItems();
    countdown.reset();
  });
});
