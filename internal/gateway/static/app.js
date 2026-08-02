(() => {
  const csrf = document.querySelector('meta[name="csrf-token"]').content;
  const mutate = async (url, options = {}) => {
    const response = await fetch(url, {
      ...options,
      credentials: "same-origin",
      headers: {"X-CSRF-Token": csrf, ...(options.headers || {})}
    });
    if (!response.ok) {
      let message = `Request failed (${response.status})`;
      try { message = (await response.json()).error || message; } catch (_) {}
      throw new Error(message);
    }
    return response;
  };

  document.querySelectorAll("[data-scan-provider]").forEach((button) => {
    button.addEventListener("click", async () => {
      button.disabled = true;
      button.textContent = "Scan queued";
      try {
        await mutate(`/api/providers/${button.dataset.scanProvider}/scan`, {method: "POST"});
        window.setTimeout(() => window.location.reload(), 700);
      } catch (error) {
        window.alert(error.message);
        button.disabled = false;
        button.textContent = "Scan";
      }
    });
  });

  document.getElementById("enroll-provider")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const form = event.currentTarget;
    const button = form.querySelector("button");
    button.disabled = true;
    button.textContent = "Enrolling…";
    try {
      const response = await mutate("/api/providers", {
        method: "POST",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify({
          displayName: form.elements.displayName.value,
          root: form.elements.root.value
        })
      });
      const provider = await response.json();
      await mutate(`/api/providers/${provider.id}/scan`, {method: "POST"});
      window.location.reload();
    } catch (error) {
      window.alert(error.message);
      button.disabled = false;
      button.textContent = "Enroll and scan";
    }
  });
})();
