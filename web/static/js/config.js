/**
 * lmon config.js
 * Configuration page logic refactored to use shared utilities from utils.js.
 */

import { getIcon, showToast, fetchJson, handleFetchError } from "./utils.js";

// These will be set by the template engine as global variables
// Example: const default_health_icon = "heart-pulse"
// Example: const default_disk_icon = "hdd"

// Function to load configuration

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

async function loadConfig() {
  try {
    const config = await fetchJson("/api/config");

    // Render disk config items
    window.diskArray = Object.entries(config.Monitoring.Disk || {}).map(
      ([name, props]) => ({
        name,
        ...props,
      }),
    );
    renderDiskConfig(window.diskArray);

    // Render system config (passing web config for dashboard title)
    window.lastLoadedInterval = config.Monitoring.Interval || 60;
    renderSystemConfig(config.Monitoring.System, config.Web);

    // Render health check config items
    window.healthArray = Object.entries(
      config.Monitoring.Healthcheck || {},
    ).map(([name, props]) => ({
      name,
      ...props,
    }));
    renderHealthConfig(window.healthArray);

    // Render webhook config
    renderWebhookConfig(config.Webhook);
  } catch (error) {
    handleFetchError(error, "Failed to load configuration");
  }
}

// Function to render disk config items
function renderDiskConfig(diskItems) {
  if (!diskItems || diskItems.length === 0) {
    document.getElementById("disk-config-items").innerHTML =
      '<div class="text-center">No disk monitors configured</div>';
    return;
  }

  let html = "";
  console.log(diskItems);
  diskItems.forEach((item) => {
    html += `
      <div class="config-item">
        <div class="d-flex justify-content-between align-items-center">
          <div>        
            ${getIcon({ icon: item.Icon, type: "disk"})}
            <strong>${item.name} (${item.Path || "(no path)"})</strong>
          </div>
          <div>
            <span>Threshold: ${
              item.Threshold !== undefined && item.Threshold !== null
                ? parseFloat(item.Threshold).toFixed(2)
                : "N/A"
            }%</span>
            <button type="button" class="btn btn-link p-0 delete-btn ms-2" data-type="disk" data-id="${item.name}" aria-label="Delete">
              <i class="bi bi-trash"></i>
            </button>
          </div>
        </div>
      </div>
    `;
  });

  document.getElementById("disk-config-items").innerHTML = html;

  // Add delete event listeners
  document.querySelectorAll('.delete-btn[data-type="disk"]').forEach((btn) => {
    btn.addEventListener("click", function () {
      const id = this.getAttribute("data-id");
      const disk = (window.diskArray || []).find((d) => d.name === id);
      let detail = id;
      if (disk) {
        detail = `${disk.name} (${disk.Path || ""})`;
      }
      deleteMonitor("disk", id, detail);
    });
  });
}

// Function to render system config
function renderSystemConfig(systemConfig, webConfig) {
  if (!systemConfig) {
    document.getElementById("system-config-items").innerHTML =
      '<div class="text-center">No system monitoring configured</div>';
    return;
  }

  const cpuConfig = systemConfig.CPU || {};
  const memoryConfig = systemConfig.Memory || {};

  document.getElementById("system-config-items").innerHTML = `
    <form id="inline-system-form">
      <div class="config-item">
        <div class="d-flex justify-content-between align-items-center">
          <div>
            ${getIcon({ icon: cpuConfig.Icon, type: "cpu"})}
            <strong>CPU Monitoring</strong>
          </div>
          <div>
            <input
              type="number"
              class="form-control"
              id="cpu-threshold-inline"
              min="1"
              max="100"
              style="width: 100px; display: inline-block;"
              value="${cpuConfig.Threshold !== undefined && cpuConfig.Threshold !== null ? cpuConfig.Threshold : ""}"
              required
            />
            <span class="ms-2">%</span>
          </div>
        </div>
      </div>
      <div class="config-item">
        <div class="d-flex justify-content-between align-items-center">
          <div>
            ${getIcon({ icon: memoryConfig.Icon, type: "memory"})}
            <strong>Memory Monitoring</strong>
          </div>
          <div>
            <input
              type="number"
              class="form-control"
              id="memory-threshold-inline"
              min="1"
              max="100"
              style="width: 100px; display: inline-block;"
              value="${memoryConfig.Threshold !== undefined && memoryConfig.Threshold !== null ? memoryConfig.Threshold : ""}"
              required
            />
            <span class="ms-2">%</span>
          </div>
        </div>
      </div>
      <div class="config-item">
        <div class="d-flex justify-content-between align-items-center">
          <div>
            <i class="bi bi-clock-history item-icon"></i>
            <strong>Refresh Interval</strong>
          </div>
          <div>
            <input
              type="number"
              class="form-control"
              id="refresh-interval-inline"
              min="1"
              style="width: 100px; display: inline-block;"
              value="${window.lastLoadedInterval !== undefined ? window.lastLoadedInterval : ""}"
              required
            />
            <span class="ms-2">seconds</span>
          </div>
        </div>
      </div>
      <div class="config-item">
        <div class="d-flex justify-content-between align-items-center">
          <div>
            <i class="bi bi-gear item-icon"></i>
            <strong>Dashboard Title</strong>
          </div>
          <div>
            <input
              type="text"
              class="form-control"
              id="dashboard-title-inline"
              style="width: 250px; display: inline-block;"
              value="${systemConfig.Title || "Monitoring Dashboard"}"
              required
            />
          </div>
        </div>
      </div>
      <div class="text-end mt-2">
        <button id="save-system-inline-btn" class="btn btn-primary">Save</button>
      </div>
    </form>
  `;

  // Add submit handler for inline system form
  const inlineForm = document.getElementById("inline-system-form");
  if (inlineForm) {
    inlineForm.addEventListener("submit", async function (e) {
      e.preventDefault();
      const cpuThreshold = parseInt(
        document.getElementById("cpu-threshold-inline").value,
        10,
      );
      const memThreshold = parseInt(
        document.getElementById("memory-threshold-inline").value,
        10,
      );
      const interval = parseInt(
        document.getElementById("refresh-interval-inline").value,
        10,
      );
      const dashboardTitle = document.getElementById(
        "dashboard-title-inline",
      ).value;

      if (
        isNaN(cpuThreshold) ||
        isNaN(memThreshold) ||
        isNaN(interval) ||
        !dashboardTitle
      ) {
        showToast("Error", "Please fill out all system settings.", true);
        return;
      }

      try {
        // Save system thresholds and title
        await fetchJson("/api/config/system", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            CPU: { Threshold: cpuThreshold, Icon: cpuConfig.Icon || "cpu" },
            Memory: {
              Threshold: memThreshold,
              Icon: memoryConfig.Icon || "speedometer",
            },
            Title: dashboardTitle,
          }),
        });
        // Save interval separately
        await fetchJson("/api/config/interval", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ Interval: interval }),
        });
        showToast("Success", "System settings updated");
        loadConfig();
      } catch (error) {
        handleFetchError(error, "Failed to update system settings");
      }
    });
  }
}

// Function to render health check config items
function renderHealthConfig(healthItems) {
  if (!healthItems || healthItems.length === 0) {
    document.getElementById("health-config-items").innerHTML =
      '<div class="text-center">No health checks configured</div>';
    return;
  }

  let html = "";
  console.log(healthItems);
  healthItems.forEach((item) => {
    html += `
      <div class="config-item">
        <div class="d-flex justify-content-between align-items-center">
          <div>
            ${getIcon({ icon: item.Icon, type: "health" })}
            <strong>${item.name}</strong>
          </div>
          <div>
            <button type="button" class="btn btn-link p-0 delete-btn" data-type="health" data-id="${item.name}" aria-label="Delete">
              <i class="bi bi-trash"></i>
            </button>
          </div>
        </div>
        <div class="mt-2">
          <small class="text-muted">${item.URL}</small>
        </div>
      </div>
    `;
  });

  document.getElementById("health-config-items").innerHTML = html;

  // Add delete event listeners
  document
    .querySelectorAll('.delete-btn[data-type="health"]')
    .forEach((btn) => {
      btn.addEventListener("click", function () {
        const id = this.getAttribute("data-id");
        const health = (window.healthArray || []).find((h) => h.name === id);
        let detail = id;
        if (health) {
          detail = `${health.name} (${health.URL || ""})`;
        }
        deleteMonitor("health", id, detail);
      });
    });
}

// Function to render webhook config
function renderWebhookConfig(webhookConfig) {
  if (!webhookConfig) {
    document.getElementById("webhook-config").innerHTML =
      '<div class="text-center">No webhook configured</div>';
    return;
  }

  document.getElementById("webhook-config").innerHTML = `
    <form id="webhook-inline-form">
      <div class="config-item">
        <div class="d-flex justify-content-between align-items-center">
          <div>
            <i class="bi bi-bell item-icon"></i>
            <strong>Webhook Notifications</strong>
          </div>
          <div>
            <span class="badge ${webhookConfig.Enabled ? "bg-success" : "bg-secondary"}">
              ${webhookConfig.Enabled ? "Enabled" : "Disabled"}
            </span>
          </div>
        </div>
        <div class="mt-2">
          <input
            type="text"
            class="form-control"
            id="webhook-url-inline"
            placeholder="Webhook URL"
            value="${webhookConfig.URL || ""}"
            style="width: 350px; display: inline-block;"
            required
          />
        </div>
        <div class="mt-2 text-end">
          ${
            webhookConfig.Enabled
              ? `<button id="webhook-update-btn" class="btn btn-primary me-2" type="submit">Update Webhook</button>
                 <button id="webhook-disable-btn" class="btn btn-secondary" type="button">Disable Webhook</button>`
              : `<button id="webhook-enable-btn" class="btn btn-primary" type="submit">Enable Webhook</button>`
          }
        </div>
      </div>
    </form>
  `;

  const webhookForm = document.getElementById("webhook-inline-form");
  const webhookUrlInput = document.getElementById("webhook-url-inline");
  const updateBtn =
    document.getElementById("webhook-update-btn") ||
    document.getElementById("webhook-enable-btn");
  const disableBtn = document.getElementById("webhook-disable-btn");

  function setWebhookLoading(isLoading) {
    if (updateBtn) updateBtn.disabled = isLoading;
    if (disableBtn) disableBtn.disabled = isLoading;
    if (webhookUrlInput) webhookUrlInput.disabled = isLoading;
  }

  if (webhookForm) {
    webhookForm.addEventListener("submit", async function (e) {
      e.preventDefault();
      const url = webhookUrlInput.value.trim();
      if (!url) {
        showToast("Error", "Webhook URL is required", true);
        return;
      }
      setWebhookLoading(true);
      try {
        await fetchJson("/api/config/webhook", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            enabled: true,
            url: url,
          }),
        });
        showToast(
          "Success",
          webhookConfig.enabled ? "Webhook updated" : "Webhook enabled",
        );
        setTimeout(() => {
          loadConfig();
          setWebhookLoading(false);
        }, 300);
      } catch (error) {
        setWebhookLoading(false);
        handleFetchError(error, "Failed to update webhook");
      }
    });
  }
  if (disableBtn) {
    disableBtn.addEventListener("click", async function () {
      const url = webhookUrlInput.value.trim();
      setWebhookLoading(true);
      try {
        await fetchJson("/api/config/webhook", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            enabled: false,
            url: url,
          }),
        });
        showToast("Success", "Webhook disabled");
        setTimeout(() => {
          loadConfig();
          setWebhookLoading(false);
        }, 300);
      } catch (error) {
        setWebhookLoading(false);
        handleFetchError(error, "Failed to disable webhook");
      }
    });
  }
}

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
    showToast("Success", data.message || "Deleted");
    loadConfig();
  } catch (error) {
    handleFetchError(error, `Failed to delete ${type} monitor`);
  }
}

// Document ready
document.addEventListener("DOMContentLoaded", function () {
  // Load configuration
  loadConfig();

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
        loadConfig();
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
        loadConfig();
        addHealthForm.reset();
      } catch (error) {
        handleFetchError(error, "Failed to add health check");
      }
    });
  }
});
