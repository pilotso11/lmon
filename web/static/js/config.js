/**
 * lmon config.js
 * Configuration page logic refactored to use shared utilities from utils.js.
 */

import { getIcon, showToast, fetchJson, handleFetchError } from "./utils.js";

// These will be set by the template engine as global variables
// Example: const default_health_icon = "heart-pulse"
// Example: const default_disk_icon = "hdd"

// --- SSR System Monitoring Form Submission ---
document.addEventListener("DOMContentLoaded", function () {
  const systemForm = document.getElementById("inline-system-form");
  if (systemForm) {
    systemForm.addEventListener("submit", async function (e) {
      e.preventDefault();
      const cpuThreshold = document.getElementById("cpu-threshold").value;
      const memoryThreshold = document.getElementById("memory-threshold").value;
      const intervalSeconds = document.getElementById("interval-seconds").value;
      const dashboardTitle = document.getElementById(
        "dashboard-title-inline",
      ).value;

      // 1. Update system config
      const systemPayload = {
        CPU: { Threshold: Number(cpuThreshold) },
        Memory: { Threshold: Number(memoryThreshold) },
        Title: dashboardTitle,
      };

      // 2. Update interval config
      const intervalPayload = { Interval: Number(intervalSeconds) };

      try {
        const [systemResp, intervalResp] = await Promise.all([
          fetch("/api/config/system", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(systemPayload),
          }),
          fetch("/api/config/interval", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(intervalPayload),
          }),
        ]);
        if (!systemResp.ok || !intervalResp.ok) {
          throw new Error("Failed to save system monitoring settings");
        }
        showToast("Success", "System monitoring settings saved.", false);
      } catch (err) {
        showToast(
          "Error",
          err.message || "Failed to save system monitoring settings",
          "danger",
        );
      }
    });
  }

  // Webhook disable button
  const webhookDisableBtn = document.getElementById("webhook-disable-btn");
  const webhookEnableBtn = document.getElementById("webhook-enable-btn");
  const webhookUpdateBtn = document.getElementById("webhook-update-btn");

  // webhook disable button
  if (webhookDisableBtn) {
    webhookDisableBtn.addEventListener("click", async function () {
      const urlInput = document.getElementById("webhook-url-inline");
      const urlValue = urlInput ? urlInput.value : "";
      try {
        const resp = await fetch("/api/config/webhook", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ Enabled: false, URL: urlValue }),
        });
        if (!resp.ok) {
          throw new Error("Failed to disable webhook");
        }
        showToast("Success", "Webhook disabled.", "success");
        // Optionally, reload the page or update the UI
        window.location.reload();
      } catch (err) {
        showToast(
          "Error",
          err.message || "Failed to disable webhook",
          "danger",
        );
      }
    });
  }
  // webhook update (and enable) button
  if (webhookUpdateBtn) {
    webhookUpdateBtn.addEventListener("click", async function () {
      const urlInput = document.getElementById("webhook-url-inline");
      const urlValue = urlInput ? urlInput.value : "";
      try {
        const resp = await fetch("/api/config/webhook", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ Enabled: true, URL: urlValue }),
        });
        if (!resp.ok) {
          throw new Error("Failed to update webhook");
        }
        showToast("Success", "Webhook updated.", "success");
        // Optionally, reload the page or update the UI
        window.location.reload();
      } catch (err) {
        showToast("Error", err.message || "Failed to update webhook", "danger");
      }
    });
  }
});

/**
 * Bootstrap Icon choices for selectors.
 */
const iconChoices = [
  { name: "cpu", icon: "cpu" },
  { name: "memory", icon: "memory" },
  { name: "sd-card", icon: "sd-card" },
  { name: "hdd", icon: "hdd" },
  { name: "hdd-network", icon: "hdd-network" },
  { name: "hdd-rack", icon: "hdd-rack" },
  { name: "device-hdd", icon: "device-hdd" },
  { name: "database", icon: "database" },
  { name: "pc-horizontal", icon: "pc-horizontal" },
  { name: "pc", icon: "pc" },
  { name: "activity", icon: "activity" },
  { name: "heart-pulse", icon: "heart-pulse" },
  { name: "speedometer", icon: "speedometer" },
  { name: "speedometer2", icon: "speedometer2" },
  { name: "bar-chart", icon: "bar-chart" },
  { name: "graph-up", icon: "graph-up" },
  { name: "router", icon: "router" },
  { name: "wifi", icon: "wifi" },
  { name: "house", icon: "house" },
  { name: "building", icon: "building" },
  { name: "lightning", icon: "lightning" },
  { name: "lightbulb", icon: "lightbulb" },
  { name: "lamp", icon: "lamp" },
  { name: "at", icon: "at" },
  { name: "battery", icon: "battery" },
  { name: "globe", icon: "globe" },
  { name: "printer", icon: "printer" },
  { name: "folder", icon: "folder" },
  { name: "shield", icon: "shield" },
  { name: "collection", icon: "collection" },
  { name: "envelope", icon: "envelope" },
  { name: "inbox", icon: "inbox" },
  { name: "people", icon: "people" },
  { name: "person-circle", icon: "person-circle" },
  { name: "webcam", icon: "webcam" },
  { name: "volume-up", icon: "volume-up" },
  { name: "voicemail", icon: "voicemail" },
  { name: "tv", icon: "tv" },
];

/**
 * Render a Bootstrap-select icon dropdown into a container.
 * @param {string} containerId - The id of the container div.
 * @param {string} selectId - The id for the <select> element.
 * @param {string} defaultIcon - The default icon name.
 */
function renderIconDropdown(containerId, selectId, defaultIcon) {
  const container = document.getElementById(containerId);
  if (!container) return;

  let optionsHtml = iconChoices
    .map(
      (choice) =>
        `<option value="${choice.icon}"${choice.icon === defaultIcon ? " selected" : ""} data-content='<i class="bi bi-${choice.icon}"></i> ${choice.name}'>
          ${choice.name}
        </option>`,
    )
    .join("");

  container.innerHTML = `
    <select class="selectpicker" id="${selectId}" data-live-search="true" data-width="100%" data-max-options="10" data-max-options-text="more...">
      ${optionsHtml}
    </select>
  `;

  // Initialize bootstrap-select
  if (window.$ && typeof window.$.fn.selectpicker === "function") {
    window.$(`#${selectId}`).selectpicker("render");
  }
}

// Render icon dropdowns on DOMContentLoaded
document.addEventListener("DOMContentLoaded", function () {
  if (typeof default_disk_icon !== "undefined") {
    renderIconDropdown("disk-icon-dropdown", "disk-icon", default_disk_icon);
  }
  if (typeof default_health_icon !== "undefined") {
    renderIconDropdown(
      "health-icon-dropdown",
      "health-icon-select",
      default_health_icon,
    );
  }
});

// No longer needed: loadConfig() is obsolete since SSR provides all necessary details for delete popups.

// No longer rendering disk config items client-side; SSR handles this.
// Only event listeners for delete buttons are needed.
document.addEventListener("DOMContentLoaded", function () {
  document.querySelectorAll(".delete-disk-btn").forEach((btn) => {
    btn.addEventListener("click", function () {
      const id = this.getAttribute("data-id");
      const detail = this.getAttribute("data-detail") || id;
      deleteMonitor("disk", id, detail);
    });
  });
});

// No longer rendering health check config items client-side; SSR handles this.
// Only event listeners for delete buttons are needed.
document.addEventListener("DOMContentLoaded", function () {
  document.querySelectorAll(".delete-health-btn").forEach((btn) => {
    btn.addEventListener("click", function () {
      const id = this.getAttribute("data-id");
      const detail = this.getAttribute("data-detail") || id;
      deleteMonitor("health", id, detail);
    });
  });
});

// Function to delete a monitor
async function deleteMonitor(type, id, detail) {
  const msg = detail
    ? `Are you sure you want to delete this ${type} monitor: ${detail}?`
    : `Are you sure you want to delete this ${type} monitor?`;
  if (!confirm(msg)) {
    return;
  }

  try {
    const data = await fetchJson(`/api/config/${type}/${id}`, {
      method: "DELETE",
    });
    window.location.reload();
    showToast("Success", data.message || "Deleted");
  } catch (error) {
    handleFetchError(error, `Failed to delete ${type} monitor`);
  }
}

// Document ready
document.addEventListener("DOMContentLoaded", function () {
  // Add disk form submission
  const addDiskForm = document.getElementById("add-disk-form");
  if (addDiskForm) {
    addDiskForm.addEventListener("submit", async function (e) {
      e.preventDefault();

      const diskName = document.getElementById("disk-name").value.trim();
      const diskPath = document.getElementById("disk-path").value.trim();
      const diskIcon = document.getElementById("disk-icon")
        ? document.getElementById("disk-icon").value.trim()
        : "";
      const diskThreshold = parseInt(
        document.getElementById("disk-threshold").value,
      );

      if (!diskName) {
        showToast("Error", "Disk name is required", true);
        return;
      }

      const diskConfig = {
        path: diskPath,
        icon: diskIcon,
        threshold: diskThreshold,
      };

      try {
        await fetchJson(`/api/config/disk/${encodeURIComponent(diskName)}`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(diskConfig),
        });
        showToast("Success", "Disk monitor added");
        window.location.reload();
        addDiskForm.reset();
      } catch (error) {
        handleFetchError(error, "Failed to add disk monitor");
      }
    });
  }

  // Add health check form submission
  const addHealthForm = document.getElementById("add-health-form");
  if (addHealthForm) {
    addHealthForm.addEventListener("submit", async function (e) {
      e.preventDefault();

      const healthNameInput = document.getElementById("health-name");
      const healthUrlInput = document.getElementById("health-url");
      const healthTimeoutInput = document.getElementById("health-timeout");
      const healthIconInput = document.getElementById("health-icon-select");
      if (
        !healthNameInput ||
        !healthUrlInput ||
        !healthTimeoutInput ||
        !healthIconInput
      ) {
        showToast("Error", "Healthcheck form fields missing", true);
        return;
      }

      const healthConfig = {
        name: healthNameInput.value,
        url: healthUrlInput.value,
        timeout: parseInt(healthTimeoutInput.value),
        icon: healthIconInput.value,
      };

      try {
        await fetchJson(
          `/api/config/health/${encodeURIComponent(healthConfig.name)}`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
            },
            body: JSON.stringify({
              url: healthConfig.url,
              icon: healthConfig.icon,
              timeout: parseInt(healthConfig.timeout),
            }),
          },
        );
        showToast("Success", "Health check added");
        window.location.reload();
        addHealthForm.reset();
      } catch (error) {
        handleFetchError(error, "Failed to add health check");
      }
    });
  }
});
