/**
 * lmon config.js
 * Configuration page logic refactored to use shared utilities from utils.js.
 */

import { showToast, fetchJson, handleFetchError } from "./utils.js";

// These will be set by the template engine as global variables
// Example: const default_health_icon = "heart-pulse"
// Example: const default_disk_icon = "hdd"

// --- SSR System Monitoring Form Submission ---
document.addEventListener("DOMContentLoaded", function () {
  // Show pending toast if present
  const pendingToast = localStorage.getItem("pendingToast");
  if (pendingToast) {
    const { title, message, type } = JSON.parse(pendingToast);
    showToast(title, message, type);
    localStorage.removeItem("pendingToast");
  }

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
        showToast("Success", "System monitoring settings saved.", "success");
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
  { name: "arrow-left-right", icon: "arrow-left-right" },
  { name: "arrow-up-down", icon: "arrow-up-down" },
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
      "monitor-icon-dropdown",
      "monitor-icon-select",
      default_health_icon,
    );
  }
  if (typeof default_docker_icon !== "undefined") {
    renderIconDropdown(
      "docker-icon-dropdown",
      "docker-icon",
      default_docker_icon,
    );
  }

  // Handle monitor type toggle
  const httpRadio = document.getElementById("monitor-type-http");
  const pingRadio = document.getElementById("monitor-type-ping");
  const targetLabel = document.getElementById("monitor-target-label");
  const targetField = document.getElementById("monitor-target");
  const timeoutLabel = document.getElementById("monitor-timeout-label");
  const timeoutField = document.getElementById("monitor-timeout");
  const respCodeRow = document.getElementById("respcode-row");
  const respCodeField = document.getElementById("monitor-respcode");
  const restartContainersRow = document.getElementById(
    "restart-containers-row",
  );
  const amberThresholdRow = document.getElementById("amber-threshold-row");
  const amberThresholdField = document.getElementById(
    "monitor-amber-threshold",
  );
  const submitButton = document.getElementById("add-monitor-button");

  function updateFormForType(type) {
    if (type === "ping") {
      // Update labels and fields for ping
      targetLabel.textContent = "Address";
      targetField.type = "text";
      targetField.placeholder = "e.g., google.com or 8.8.8.8";
      targetField.setAttribute("aria-label", "Ping address (IP or hostname)");
      timeoutLabel.textContent = "Timeout (ms)";
      timeoutField.value = "100";
      timeoutField.min = "100";
      timeoutField.max = "30000";
      timeoutField.setAttribute("aria-label", "Ping timeout in milliseconds");
      respCodeRow.style.display = "none";
      restartContainersRow.style.display = "none";
      amberThresholdRow.style.display = "flex";
      amberThresholdField.required = true;
      submitButton.textContent = "Add Ping Monitor";

      // Update icon dropdown to use ping icon if available
      if (typeof default_ping_icon !== "undefined") {
        renderIconDropdown(
          "monitor-icon-dropdown",
          "monitor-icon-select",
          default_ping_icon,
        );
      }
    } else {
      // Update labels and fields for HTTP
      targetLabel.textContent = "URL";
      targetField.type = "url";
      targetField.placeholder = "";
      targetField.setAttribute("aria-label", "Health check URL");
      timeoutLabel.textContent = "Timeout (seconds)";
      timeoutField.value = "10";
      timeoutField.min = "1";
      timeoutField.removeAttribute("max");
      timeoutField.setAttribute(
        "aria-label",
        "Health check timeout in seconds",
      );
      respCodeRow.style.display = "flex";
      respCodeField.value = "200";
      respCodeField.min = "100";
      respCodeField.max = "599";
      restartContainersRow.style.display = "flex";
      amberThresholdRow.style.display = "none";
      amberThresholdField.required = false;
      submitButton.textContent = "Add Health Check";

      // Update icon dropdown to use health icon if available
      if (typeof default_health_icon !== "undefined") {
        renderIconDropdown(
          "monitor-icon-dropdown",
          "monitor-icon-select",
          default_health_icon,
        );
      }
    }
  }

  // Initialize form based on default selection
  if (httpRadio && httpRadio.checked) {
    updateFormForType("http");
  } else if (pingRadio && pingRadio.checked) {
    updateFormForType("ping");
  }

  // Add change listeners for radio buttons
  if (httpRadio) {
    httpRadio.addEventListener("change", function () {
      if (this.checked) {
        updateFormForType("http");
      }
    });
  }

  if (pingRadio) {
    pingRadio.addEventListener("change", function () {
      if (this.checked) {
        updateFormForType("ping");
      }
    });
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

// No longer rendering ping config items client-side; SSR handles this.
// Only event listeners for delete buttons are needed.
document.addEventListener("DOMContentLoaded", function () {
  document.querySelectorAll(".delete-ping-btn").forEach((btn) => {
    btn.addEventListener("click", function () {
      const id = this.getAttribute("data-id");
      const detail = this.getAttribute("data-detail") || id;
      deleteMonitor("ping", id, detail);
    });
  });
});

// No longer rendering Docker config items client-side; SSR handles this.
// Only event listeners for delete buttons are needed.
document.addEventListener("DOMContentLoaded", function () {
  document.querySelectorAll(".delete-docker-btn").forEach((btn) => {
    btn.addEventListener("click", function () {
      const id = this.getAttribute("data-id");
      const detail = this.getAttribute("data-detail") || id;
      deleteMonitor("docker", id, detail);
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
    // Persist toast info before reload
    localStorage.setItem(
      "pendingToast",
      JSON.stringify({
        title: "Success",
        message: data.message || "Deleted",
        type: "success",
      }),
    );
    window.location.reload();
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
        // Persist toast info before reload
        localStorage.setItem(
          "pendingToast",
          JSON.stringify({
            title: "Success",
            message: "Disk monitor added",
            type: "success",
          }),
        );
        window.location.reload();
      } catch (error) {
        handleFetchError(error, "Failed to add disk monitor");
      }
    });
  }

  // Add unified monitor form submission
  const addMonitorForm = document.getElementById("add-monitor-form");
  if (addMonitorForm) {
    addMonitorForm.addEventListener("submit", async function (e) {
      e.preventDefault();

      const nameInput = document.getElementById("monitor-name");
      const targetInput = document.getElementById("monitor-target");
      const timeoutInput = document.getElementById("monitor-timeout");
      const respCodeInput = document.getElementById("monitor-respcode");
      const iconInput = document.getElementById("monitor-icon-select");
      const typeRadio = document.querySelector(
        'input[name="monitor-type"]:checked',
      );

      if (
        !nameInput ||
        !targetInput ||
        !timeoutInput ||
        !iconInput ||
        !typeRadio
      ) {
        showToast("Error", "Monitor form fields missing", "danger");
        return;
      }

      const monitorType = typeRadio.value;
      const name = nameInput.value.trim();
      const target = targetInput.value.trim();
      const timeout = parseInt(timeoutInput.value);
      const icon = iconInput.value;

      if (!name) {
        showToast("Error", "Monitor name is required", "danger");
        return;
      }

      try {
        if (monitorType === "ping") {
          const amberThresholdInput = document.getElementById(
            "monitor-amber-threshold",
          );
          if (!amberThresholdInput) {
            showToast("Error", "Amber threshold field missing", "danger");
            return;
          }
          const amberThreshold = parseInt(amberThresholdInput.value);

          await fetchJson(`/api/config/ping/${encodeURIComponent(name)}`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              address: target,
              timeout: timeout,
              amberThreshold: amberThreshold,
              icon: icon,
            }),
          });

          localStorage.setItem(
            "pendingToast",
            JSON.stringify({
              title: "Success",
              message: "Ping monitor added",
              type: "success",
            }),
          );
        } else {
          const respCode = parseInt(respCodeInput.value);
          if (respCode < 100 || respCode > 599) {
            showToast(
              "Error",
              "Response code must be between 100 and 599",
              "danger",
            );
            return;
          }

          // Get restart containers value (optional)
          const restartContainersInput = document.getElementById(
            "monitor-restart-containers",
          );
          const restartContainers = restartContainersInput
            ? restartContainersInput.value.trim()
            : "";

          // HTTP health check
          await fetchJson(`/api/config/health/${encodeURIComponent(name)}`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              url: target,
              timeout: timeout,
              respcode: respCode,
              icon: icon,
              restart_containers: restartContainers,
            }),
          });

          localStorage.setItem(
            "pendingToast",
            JSON.stringify({
              title: "Success",
              message: "Health check added",
              type: "success",
            }),
          );
        }

        window.location.reload();
      } catch (error) {
        handleFetchError(
          error,
          `Failed to add ${monitorType === "ping" ? "ping monitor" : "health check"}`,
        );
      }
    });
  }

  // Add Docker monitor form submission
  const addDockerForm = document.getElementById("add-docker-form");
  if (addDockerForm) {
    addDockerForm.addEventListener("submit", async function (e) {
      e.preventDefault();

      const dockerName = document.getElementById("docker-name").value.trim();
      const dockerContainers = document
        .getElementById("docker-containers")
        .value.trim();
      const dockerIcon = document.getElementById("docker-icon")
        ? document.getElementById("docker-icon").value.trim()
        : "";
      const dockerThreshold = parseInt(
        document.getElementById("docker-threshold").value,
      );

      if (!dockerName) {
        showToast("Error", "Docker monitor name is required", "danger");
        return;
      }

      if (!dockerContainers) {
        showToast("Error", "Container names are required", "danger");
        return;
      }

      const dockerConfig = {
        containers: dockerContainers,
        icon: dockerIcon,
        threshold: dockerThreshold,
      };

      try {
        await fetchJson(
          `/api/config/docker/${encodeURIComponent(dockerName)}`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
            },
            body: JSON.stringify(dockerConfig),
          },
        );
        // Persist toast info before reload
        localStorage.setItem(
          "pendingToast",
          JSON.stringify({
            title: "Success",
            message: "Docker monitor added",
            type: "success",
          }),
        );
        window.location.reload();
      } catch (error) {
        handleFetchError(error, "Failed to add Docker monitor");
      }
    });
  }
});
