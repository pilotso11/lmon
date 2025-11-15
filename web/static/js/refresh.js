document.addEventListener("DOMContentLoaded", function () {
  const refreshBtn = document.getElementById("refresh-btn");
  const countdownElement = document.getElementById("refresh-countdown");
  // Get the interval from the button's data attribute
  if (!refreshBtn) {
    // If the refresh button is not present, do nothing
    return;
  }
  const refreshInterval =
    parseInt(refreshBtn.getAttribute("data-refresh-interval")) || 60;
  let remaining = refreshInterval;

  function updateCountdown() {
    if (countdownElement) {
      countdownElement.textContent = ` (${remaining}s)`;
    }
  }

  updateCountdown();
  setInterval(function () {
    remaining--;
    updateCountdown();
    if (remaining <= 0) {
      window.location.reload();
    }
  }, 1000);

  if (refreshBtn) {
    refreshBtn.addEventListener("click", function () {
      window.location.reload();
    });
  }

  // Handle Docker container restart buttons
  document.querySelectorAll(".restart-docker-btn").forEach((btn) => {
    btn.addEventListener("click", async function (e) {
      e.preventDefault();
      e.stopPropagation();

      const monitorId = this.getAttribute("data-id");
      // Try to find the display name in both desktop and mobile layouts
      let displayNameElem = this.closest(".item-container")?.querySelector('[aria-label*="Name"]');
      if (!displayNameElem) {
        // Fallback: try to find a nearby element with the aria-label in mobile view
        displayNameElem = this.parentElement?.querySelector('[aria-label*="Name"]');
      }
      if (!displayNameElem) {
        // As a last resort, search the whole document (should not usually be needed)
        displayNameElem = document.querySelector('[aria-label*="Name"]');
      }
      const displayName = displayNameElem ? displayNameElem.textContent.trim() : "this monitor";

      if (
        !confirm(
          `Are you sure you want to restart containers for ${displayName}?`,
        )
      ) {
        return;
      }

      // Disable button during restart
      const originalHTML = this.innerHTML;
      this.disabled = true;
      this.innerHTML =
        '<i class="bi bi-hourglass-split" aria-hidden="true"></i>';

      try {
        // Extract the monitor name from the ID (format: docker_name)
        const monitorName = monitorId.replace(/^docker_/, "");
        const response = await fetch(
          `/api/action/docker/${encodeURIComponent(monitorName)}/restart`,
          {
            method: "POST",
          },
        );

        if (!response.ok) {
          const errorText = await response.text();
          throw new Error(errorText || `HTTP error ${response.status}`);
        }

        // Show success message
        alert("Containers restarted successfully");

        // Refresh the page to show updated status
        window.location.reload();
      } catch (error) {
        alert(`Failed to restart containers: ${error.message}`);
        // Re-enable button on error
        this.disabled = false;
        this.innerHTML = originalHTML;
      }
    });
  });

  // Handle Healthcheck container restart buttons
  document.querySelectorAll(".restart-healthcheck-btn").forEach((btn) => {
    btn.addEventListener("click", async function (e) {
      e.preventDefault();
      e.stopPropagation();

      const monitorId = this.getAttribute("data-id");
      const displayName = this.closest(".item-container")
        .querySelector('[aria-label*="Name"]')
        .textContent.trim();

      if (
        !confirm(
          `Are you sure you want to restart containers for ${displayName}?`,
        )
      ) {
        return;
      }

      // Disable button during restart
      const originalHTML = this.innerHTML;
      this.disabled = true;
      this.innerHTML =
        '<i class="bi bi-hourglass-split" aria-hidden="true"></i>';

      try {
        // Extract the monitor name from the ID (format: health_name)
        const monitorName = monitorId.replace(/^health_/, "");
        const response = await fetch(
          `/api/action/healthcheck/${encodeURIComponent(monitorName)}/restart`,
          {
            method: "POST",
          },
        );

        if (!response.ok) {
          const errorText = await response.text();
          throw new Error(errorText || `HTTP error ${response.status}`);
        }

        // Show success message
        alert("Containers restarted successfully");

        // Refresh the page to show updated status
        window.location.reload();
      } catch (error) {
        alert(`Failed to restart containers: ${error.message}`);
        // Re-enable button on error
        this.disabled = false;
        this.innerHTML = originalHTML;
      }
    });
  });
});
