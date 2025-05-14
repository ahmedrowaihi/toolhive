const UI = {
  get elements() {
    return new Proxy(
      {},
      {
        get: (target, prop) => {
          return document.getElementById(prop);
        },
      }
    );
  },

  loader: (() => {
    const loader = document.createElement("div");
    loader.className = "loader";
    loader.innerHTML = `
      <div class="loader-content">
        <div class="spinner"></div>
        <p class="loader-text">Loading...</p>
      </div>
    `;
    document.body.appendChild(loader);
    return loader;
  })(),

  showLoader(text = "Loading...") {
    const main = document.querySelector("main");
    if (main) {
      main.setAttribute("aria-busy", "true");
      main.setAttribute("aria-label", text);
    }
  },

  hideLoader() {
    const main = document.querySelector("main");
    if (main) {
      main.removeAttribute("aria-busy");
      main.removeAttribute("aria-label");
    }
  },

  async withLoadingState(text, action) {
    this.showLoader(text);
    try {
      await action();
    } catch (error) {
      console.error("Operation failed:", error);
      throw error;
    } finally {
      this.hideLoader();
    }
  },

  closeModal() {
    const modal = this.elements.newServerModal;
    if (modal) {
      modal.close();
    }
    const form = this.elements.newServerForm;
    if (form) {
      form.reset();
    }
    const outputContainer = document.getElementById("commandOutput");
    if (outputContainer) {
      outputContainer.style.display = "none";
      outputContainer.textContent = "";
    }
  },

  closeSettingsModal() {
    const modal = this.elements.settingsModal;
    if (modal) {
      modal.close();
    }
  },

  showSettingsModal() {
    const modal = this.elements.settingsModal;
    if (modal) {
      modal.showModal();
    }
  },

  showCommandOutput(output, isSuccess) {
    let outputContainer = document.getElementById("commandOutput");
    if (!outputContainer) {
      outputContainer = document.createElement("div");
      outputContainer.id = "commandOutput";
      outputContainer.className = "command-output";
      const form = this.elements.newServerForm;
      if (form) {
        form.appendChild(outputContainer);
      }
    }

    outputContainer.textContent = output;
    outputContainer.style.display = "block";
    outputContainer.className = `command-output ${
      isSuccess ? "success" : "error"
    }`;
  },
};

const API = {
  getHeaders() {
    const headers = { "Content-Type": "application/json" };
    const token = UI.elements.authToken?.value?.trim();
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
    return headers;
  },

  async handleResponse(response) {
    const data = await response.json();

    if (!response.ok) {
      if (response.status === 401) {
        const authToken = UI.elements.authToken;
        if (authToken) {
          authToken.value = "";
        }
        UI.showSettingsModal();
        throw new Error("Authentication required. Please enter a valid token.");
      }
      throw new Error(data.error || response.statusText);
    }

    if (!data.success) {
      throw new Error(data.error || "Operation failed");
    }

    return {
      ...data,
      data: data.data ?? [],
    };
  },

  async runCommand(command) {
    const response = await fetch("/api/command", {
      method: "POST",
      headers: this.getHeaders(),
      body: JSON.stringify({ command }),
    });
    const data = await this.handleResponse(response);
    return data.data;
  },

  async getServers() {
    const response = await fetch("/api/servers", {
      headers: this.getHeaders(),
    });
    const data = await this.handleResponse(response);
    return data.data;
  },

  async stopServer(name) {
    const response = await fetch(`/api/servers/${name}/stop`, {
      method: "POST",
      headers: this.getHeaders(),
    });
    const data = await this.handleResponse(response);
    return data.data;
  },

  async searchRegistry(query) {
    const response = await fetch(
      `/api/registry/search?q=${encodeURIComponent(query)}`,
      { headers: this.getHeaders() }
    );
    const data = await this.handleResponse(response);
    return data.data;
  },

  async runFromRegistry(name) {
    const response = await fetch("/api/servers", {
      method: "POST",
      headers: this.getHeaders(),
      body: JSON.stringify({ name }),
    });
    const data = await this.handleResponse(response);
    return data.data;
  },
};

const ServerManager = {
  isRefreshing: false,
  refreshTimeout: null,
  isAutoRefreshEnabled: true,

  async refreshList() {
    if (this.isRefreshing || UI.loader.style.display === "flex") return;

    this.isRefreshing = true;
    try {
      const servers = await API.getServers();
      this.displayServers(servers);
    } catch (error) {
      console.error("Error fetching servers:", error);
      UI.elements.serverList.innerHTML = `<article><p class="error">Error: ${error.message}</p></article>`;
    } finally {
      this.isRefreshing = false;
    }
  },

  startAutoRefresh() {
    if (this.refreshTimeout) clearTimeout(this.refreshTimeout);
    if (!this.isAutoRefreshEnabled) return;

    const interval = parseInt(UI.elements.refreshInterval?.value || "5") * 1000;
    this.refreshTimeout = setTimeout(() => {
      this.refreshList().finally(() => {
        if (UI.loader.style.display !== "flex") {
          this.startAutoRefresh();
        }
      });
    }, interval);
  },

  toggleAutoRefresh() {
    this.isAutoRefreshEnabled = !this.isAutoRefreshEnabled;
    const toggleBtn = UI.elements.toggleRefresh;
    if (toggleBtn) {
      toggleBtn.textContent = this.isAutoRefreshEnabled ? "Pause" : "Resume";
    }

    if (this.isAutoRefreshEnabled) {
      this.startAutoRefresh();
    } else if (this.refreshTimeout) {
      clearTimeout(this.refreshTimeout);
    }
  },

  updateRefreshInterval() {
    if (this.isAutoRefreshEnabled) {
      this.startAutoRefresh();
    }
  },

  displayServers(servers) {
    UI.elements.serverList.innerHTML = "";

    if (!servers || servers.length === 0) {
      UI.elements.serverList.innerHTML =
        "<article><p>No servers running</p></article>";
      return;
    }

    servers.forEach((server) => {
      const card = document.createElement("article");
      card.className = "server-card";
      card.innerHTML = `
        <header>
          <hgroup>
            <h3>${server.name || "Unnamed Server"}</h3>
            <h4>${server.image || "No image"}</h4>
          </hgroup>
        </header>
        <div>
          <p><strong>State:</strong> ${server.state || "unknown"}</p>
          <p><strong>Transport:</strong> ${server.transport || "unknown"}</p>
          <p><strong>Port:</strong> ${server.port || "N/A"}</p>
          <p><strong>URL:</strong> ${
            server.url
              ? `<a href="${server.url}" target="_blank">${server.url}</a>`
              : "N/A"
          }</p>
        </div>
        <footer>
          <button onclick="ServerManager.stopServer('${
            server.name
          }')" class="outline">Stop</button>
        </footer>
      `;
      UI.elements.serverList.appendChild(card);
    });
  },

  async stopServer(name) {
    if (!confirm("Are you sure you want to stop this server?")) return;

    try {
      await UI.withLoadingState("Stopping server...", async () => {
        await API.stopServer(name);
        await this.refreshList();
      });
    } catch (error) {
      alert("Error stopping server: " + error.message);
    }
  },

  saveSettings() {
    const settings = {
      refreshInterval: UI.elements.refreshInterval?.value || "5",
      isAutoRefreshEnabled: this.isAutoRefreshEnabled,
      authToken: UI.elements.authToken?.value || "",
    };
    localStorage.setItem("toolhiveSettings", JSON.stringify(settings));
  },

  loadSettings() {
    const settings = JSON.parse(
      localStorage.getItem("toolhiveSettings") || "{}"
    );

    const refreshInterval = UI.elements.refreshInterval;
    if (refreshInterval && settings.refreshInterval) {
      refreshInterval.value = settings.refreshInterval;
    }

    const authToken = UI.elements.authToken;
    if (authToken && settings.authToken) {
      authToken.value = settings.authToken;
    }

    if (settings.isAutoRefreshEnabled !== undefined) {
      this.isAutoRefreshEnabled = settings.isAutoRefreshEnabled;
      const toggleBtn = UI.elements.toggleRefresh;
      if (toggleBtn) {
        toggleBtn.textContent = this.isAutoRefreshEnabled ? "Pause" : "Resume";
      }
    }
  },
};

const RegistryManager = {
  async search() {
    const query = UI.elements.registrySearchInput.value.trim();
    if (!query) {
      alert("Please enter a search query");
      return;
    }

    try {
      await UI.withLoadingState("Searching registry...", async () => {
        const servers = await API.searchRegistry(query);
        this.displayResults(servers);
      });
    } catch (error) {
      console.error("Error searching registry:", error);
      alert("Failed to search registry: " + error.message);
    }
  },

  displayResults(servers) {
    UI.elements.registryList.innerHTML = "";

    if (!servers || servers.length === 0) {
      UI.elements.registryList.innerHTML =
        "<article><p>No MCPs found</p></article>";
      return;
    }

    servers.forEach((server) => {
      const card = document.createElement("article");
      card.className = "server-card";
      card.innerHTML = `
        <header>
          <hgroup>
            <h3>${server.name || "Unnamed Server"}</h3>
            <h4>${server.image || "No image"}</h4>
          </hgroup>
        </header>
        <div>
          <p>${server.description || "No description available"}</p>
          <p><strong>Transport:</strong> ${server.transport || "unknown"}</p>
          <p><strong>Tags:</strong> ${
            server.tags && server.tags.length > 0
              ? server.tags.join(", ")
              : "none"
          }</p>
        </div>
        <footer>
          <button onclick="RegistryManager.runFromRegistry('${
            server.name
          }')">Run</button>
        </footer>
      `;
      UI.elements.registryList.appendChild(card);
    });
  },

  async runFromRegistry(name) {
    try {
      await UI.withLoadingState("Starting server...", async () => {
        await API.runFromRegistry(name);
        await ServerManager.refreshList();

        const searchInput = UI.elements.registrySearchInput;
        if (searchInput) {
          searchInput.value = "";
        }
        const registryList = UI.elements.registryList;
        if (registryList) {
          registryList.innerHTML = "";
        }
      });
    } catch (error) {
      console.error("Error running server:", error);
      alert("Failed to run server: " + error.message);
    }
  },
};

const EventHandlers = {
  handleSearch(e) {
    const searchTerm = e.target.value.toLowerCase();
    const serverList = UI.elements.serverList;
    if (!serverList) return;

    const cards = serverList.getElementsByTagName("article");
    for (let card of cards) {
      const text = card.textContent.toLowerCase();
      card.style.display = text.includes(searchTerm) ? "" : "none";
    }
  },

  async handleNewServer(e) {
    e.preventDefault();
    const command = UI.elements.serverName.value.trim();
    if (!command) return;

    try {
      await UI.withLoadingState("Running command...", async () => {
        const response = await API.runCommand(command);
        UI.showCommandOutput(response, true);
      });
    } catch (error) {
      UI.showCommandOutput(error.message, false);
    }
  },

  handleSettingsSubmit(e) {
    e.preventDefault();
    ServerManager.saveSettings();
    UI.closeSettingsModal();
  },
};

function initialize() {
  ServerManager.loadSettings();

  const addEventListener = (element, event, handler) => {
    if (element) {
      element.addEventListener(event, handler);
    }
  };

  addEventListener(UI.elements.newServerBtn, "click", () => {
    const modal = UI.elements.newServerModal;
    if (modal) modal.showModal();
  });

  addEventListener(UI.elements.settingsBtn, "click", () => {
    UI.showSettingsModal();
  });

  addEventListener(
    UI.elements.newServerForm,
    "submit",
    EventHandlers.handleNewServer
  );
  addEventListener(
    UI.elements.settingsForm,
    "submit",
    EventHandlers.handleSettingsSubmit
  );
  addEventListener(
    UI.elements.searchInput,
    "input",
    EventHandlers.handleSearch
  );
  addEventListener(UI.elements.searchBtn, "click", () =>
    RegistryManager.search()
  );

  addEventListener(UI.elements.registrySearchInput, "keypress", (e) => {
    if (e.key === "Enter") RegistryManager.search();
  });

  addEventListener(UI.elements.toggleRefresh, "click", () => {
    ServerManager.toggleAutoRefresh();
  });

  addEventListener(UI.elements.refreshInterval, "change", () => {
    ServerManager.updateRefreshInterval();
  });

  ServerManager.refreshList().then(() => {
    ServerManager.startAutoRefresh();
  });
}

initialize();
