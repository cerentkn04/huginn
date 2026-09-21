export function getToken() {
  return localStorage.getItem("huginn_token") || "";
}

export function setToken(token) {
  localStorage.setItem("huginn_token", token);
}

export function clearToken() {
  localStorage.removeItem("huginn_token");
}

export function authFetch(url, options = {}) {
  const headers = { ...(options.headers || {}), Authorization: `Bearer ${getToken()}` };
  return fetch(url, { ...options, headers });
}

export function authEventSource(url) {
  const sep = url.includes("?") ? "&" : "?";
  return new EventSource(`${url}${sep}token=${encodeURIComponent(getToken())}`);
}
