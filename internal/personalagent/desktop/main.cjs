const { app, BrowserWindow, dialog, session, shell } = require("electron");
const { spawn } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");
const { defaultWorkspace, isSafeExternalURL, parseAgentURL } = require("./runtime.cjs");

const STARTUP_TIMEOUT_MS = 25_000;
const SHUTDOWN_TIMEOUT_MS = 8_000;
const MAX_LOG_BYTES = 8_192;

let backendProcess = null;
let mainWindow = null;
let agentOrigin = "";
let isShuttingDown = false;

if (process.platform === "linux") {
  app.commandLine.appendSwitch("no-sandbox");
}

app.setName("Cockpit Personal Agent");

if (!app.requestSingleInstanceLock()) {
  app.quit();
} else {
  app.on("second-instance", () => {
    if (mainWindow && !mainWindow.isDestroyed()) {
      if (mainWindow.isMinimized()) mainWindow.restore();
      mainWindow.focus();
    }
  });

  app.whenReady().then(startApplication).catch((error) => {
    dialog.showErrorBox("Cockpit Personal Agent", error.message);
    app.quit();
  });
}

async function startApplication() {
  try {
    const agentURL = await startBackend();
    agentOrigin = new URL(agentURL).origin;
    configurePermissions();
    await createWindow(agentURL);
  } catch (error) {
    dialog.showErrorBox("Não foi possível iniciar o Cockpit", error.message);
    app.quit();
  }
}

function startBackend() {
  const workspace = defaultWorkspace(app.getPath("home"));
  const binaryPath = backendBinaryPath();
  fs.mkdirSync(workspace, { recursive: true, mode: 0o700 });

  if (!fs.existsSync(binaryPath)) {
    throw new Error(`O serviço Cockpit não foi encontrado:\n${binaryPath}`);
  }

  const preferredPort = process.env.COCKPIT_AGENT_PORT || "0";
  backendProcess = spawn(
    binaryPath,
    ["agent", "--workspace", workspace, "--port", preferredPort],
    {
      cwd: workspace,
      env: process.env,
      stdio: ["ignore", "pipe", "pipe"],
    },
  );

  return waitForBackendStart(backendProcess);
}

function waitForBackendStart(child) {
  return new Promise((resolve, reject) => {
    const startup = {
      output: "",
      error: "",
      settled: false,
      ready: false,
      timeout: null,
      resolve,
      reject,
    };

    startup.timeout = setTimeout(() => {
      finishBackendStart(startup, new Error(`O serviço local não iniciou a tempo.\n${startup.error}`));
      child.kill("SIGTERM");
    }, STARTUP_TIMEOUT_MS);

    child.stdout.on("data", (chunk) => collectBackendOutput(startup, chunk));
    child.stderr.on("data", (chunk) => collectBackendError(startup, chunk));
    child.once("error", (error) => handleBackendSpawnError(startup, error));
    child.once("exit", (code, signal) => handleBackendExit(child, startup, code, signal));
  });
}

function collectBackendOutput(startup, chunk) {
  startup.output = appendLog(startup.output, chunk);
  try {
    const url = parseAgentURL(startup.output);
    if (url) finishBackendStart(startup, null, url);
  } catch (error) {
    finishBackendStart(startup, error);
  }
}

function collectBackendError(startup, chunk) {
  startup.error = appendLog(startup.error, chunk);
}

function appendLog(current, chunk) {
  return `${current}${chunk.toString()}`.slice(-MAX_LOG_BYTES);
}

function handleBackendSpawnError(startup, error) {
  finishBackendStart(startup, new Error(`Não foi possível executar o serviço local.\n${error.message}`));
}

function handleBackendExit(child, startup, code, signal) {
  const detail = startup.error ? `\n\n${startup.error}` : "";
  if (!startup.settled) {
    finishBackendStart(
      startup,
      new Error(`O serviço local encerrou antes de iniciar (${exitReason(code, signal)}).${detail}`),
    );
  } else if (startup.ready && !isShuttingDown) {
    showBackendExit(code, signal, detail);
  }
  if (child === backendProcess) backendProcess = null;
}

function finishBackendStart(startup, error, url) {
  if (startup.settled) return;
  startup.settled = true;
  startup.ready = !error;
  clearTimeout(startup.timeout);
  if (error) startup.reject(error);
  else startup.resolve(url);
}

function backendBinaryPath() {
  const executable = process.platform === "win32" ? "cockpit.exe" : "cockpit";
  if (app.isPackaged) return path.join(process.resourcesPath, "bin", executable);
  return path.join(__dirname, "resources", "bin", executable);
}

function createWindow(agentURL) {
  mainWindow = new BrowserWindow({
    width: 1440,
    height: 920,
    minWidth: 900,
    minHeight: 640,
    show: false,
    autoHideMenuBar: true,
    backgroundColor: "#101311",
    icon: path.join(__dirname, "assets", "cockpit-512.png"),
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      webSecurity: true,
    },
  });

  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    if (isSafeExternalURL(url)) {
      void shell.openExternal(url).catch((error) => {
        dialog.showErrorBox("Não foi possível abrir o link", error.message);
      });
    }
    return { action: "deny" };
  });
  mainWindow.webContents.on("will-navigate", (event, targetURL) => {
    if (!isAllowedAgentURL(targetURL)) event.preventDefault();
  });
  mainWindow.webContents.on("will-redirect", (event, targetURL) => {
    if (!isAllowedAgentURL(targetURL)) event.preventDefault();
  });
  mainWindow.once("ready-to-show", () => mainWindow.show());
  mainWindow.on("closed", () => {
    mainWindow = null;
  });

  return mainWindow.loadURL(agentURL);
}

function isAllowedAgentURL(value) {
  try {
    return new URL(value).origin === agentOrigin;
  } catch {
    return false;
  }
}

function configurePermissions() {
  session.defaultSession.setPermissionRequestHandler((webContents, permission, callback, details) => {
    const localOrigin = isAllowedAgentURL(webContents.getURL());
    const audioOnly = details?.mediaTypes?.includes("audio") && !details.mediaTypes.includes("video");
    callback(Boolean(permission === "media" && localOrigin && audioOnly));
  });
}

function showBackendExit(code, signal, detail) {
  const exitMessage = `O serviço local encerrou (${exitReason(code, signal)}).`;
  const message = detail ? `${exitMessage}${detail}` : exitMessage;
  if (mainWindow && !mainWindow.isDestroyed()) {
    void dialog.showMessageBox(mainWindow, {
      type: "error",
      title: "Serviço do Cockpit encerrado",
      message,
    }).finally(() => app.quit());
    return;
  }
  dialog.showErrorBox("Serviço do Cockpit encerrado", message);
  app.quit();
}

function exitReason(code, signal) {
  return signal ? `sinal ${signal}` : `código ${code}`;
}

function stopBackend() {
  if (!backendProcess || backendProcess.exitCode !== null || backendProcess.signalCode !== null) {
    return Promise.resolve();
  }

  return new Promise((resolve) => {
    let finished = false;
    const finish = () => {
      if (finished) return;
      finished = true;
      clearTimeout(shutdownTimeout);
      resolve();
    };

    const shutdownTimeout = setTimeout(() => {
      backendProcess.kill("SIGKILL");
      finish();
    }, SHUTDOWN_TIMEOUT_MS);

    backendProcess.once("exit", finish);
    backendProcess.kill("SIGTERM");
  });
}

app.on("window-all-closed", () => app.quit());
app.on("before-quit", (event) => {
  if (isShuttingDown) return;
  event.preventDefault();
  isShuttingDown = true;
  void stopBackend().finally(() => app.quit());
});
