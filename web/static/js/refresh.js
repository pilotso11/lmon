document.addEventListener("DOMContentLoaded", function () {
  const refreshBtn = document.getElementById("refresh-btn");
  const countdownElement = document.getElementById("refresh-countdown");
  // Get the interval from the button's data attribute
  const refreshInterval =
    parseInt(refreshBtn.getAttribute("data-refresh-interval")) || 60;
  let remaining = refreshInterval;

  function updateCountdown() {
    if (countdownElement) {
      countdownElement.textContent = ` (${remaining}s)`;
    }
  }

  updateCountdown();
  const intervalId = setInterval(function () {
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
});
