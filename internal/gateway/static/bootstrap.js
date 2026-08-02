(() => {
  const status = document.getElementById("bootstrap-status");
  const token = window.location.hash.slice(1);
  window.history.replaceState(null, "", "/bootstrap");
  if (!token) {
    status.textContent = "The launch token is missing. Restart SAMPO and open the new launch link.";
    return;
  }
  fetch("/session/bootstrap", {
    method: "POST",
    credentials: "same-origin",
    headers: {"Content-Type": "application/json"},
    body: JSON.stringify({token})
  }).then((response) => {
    if (!response.ok) throw new Error("session rejected");
    window.location.replace("/");
  }).catch(() => {
    status.textContent = "The private session could not be established. Restart SAMPO and try again.";
  });
})();
