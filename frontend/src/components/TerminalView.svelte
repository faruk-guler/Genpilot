<script>
  import { onMount, onDestroy } from "svelte";
  import { Terminal } from "xterm";
  import { FitAddon } from "xterm-addon-fit";
  import { ResizeTerminal, WriteToTerminal } from "../../wailsjs/go/main/App";
  import { EventsOn, EventsOff } from "../../wailsjs/runtime/runtime";
  import "xterm/css/xterm.css";

  export let connected = false;
  export let sessionId = "";
  // Callback to initiate connection
  export let onConnect = (user, pass, id) => {};

  let term;
  let fitAddon;
  let termDiv;
  let resizeListener;
  let cleanupData;
  let cleanupDisconnect;

  // Login State Machine
  let loginState = "disconnected"; // disconnected, login_user, login_pass, connected
  let inputBuffer = "";
  let username = "";
  let password = "";

  const terminalThemes = {
    genpilot: {
      background: "#09090b",
      foreground: "#e4e4e7",
      cursor: "#a1a1aa",
      black: "#09090b",
      red: "#ef4444",
      green: "#10b981",
      yellow: "#f59e0b",
      blue: "#6366f1",
      magenta: "#d946ef",
      cyan: "#06b6d4",
      white: "#e4e4e7",
    },
    putty: {
      background: "#000000",
      foreground: "#bbbbbb",
      cursor: "#bbbbbb",
      black: "#000000",
      red: "#bb0000",
      green: "#00bb00",
      yellow: "#bbbb00",
      blue: "#0000bb",
      magenta: "#bb00bb",
      cyan: "#00bbbb",
      white: "#bbbbbb",
      brightBlack: "#555555",
      brightRed: "#ff5555",
      brightGreen: "#55ff55",
      brightYellow: "#ffff55",
      brightBlue: "#5555ff",
      brightMagenta: "#ff55ff",
      brightCyan: "#55ffff",
      brightWhite: "#ffffff",
    },
    dracula: {
      background: "#282a36",
      foreground: "#f8f8f2",
      cursor: "#f8f8f2",
      black: "#21222c",
      red: "#ff5555",
      green: "#50fa7b",
      yellow: "#f1fa8c",
      blue: "#6272a4",
      magenta: "#ff79c6",
      cyan: "#8be9fd",
      white: "#f8f8f2",
    },
    solarized: {
      background: "#002b36",
      foreground: "#839496",
      cursor: "#93a1a1",
      black: "#073642",
      red: "#dc322f",
      green: "#859900",
      yellow: "#b58900",
      blue: "#268bd2",
      magenta: "#d33682",
      cyan: "#2aa198",
      white: "#eee8d5",
    },
  };

  let activeTheme = "genpilot";
  let fontSize = 14;

  function promptLogin() {
    term.reset();
    term.write(`Login as: `);
    loginState = "login_user";
    inputBuffer = "";
  }

  function handleInput(key) {
    if (loginState === "connected") return; // Let xterm handle it via onData

    const charCode = key.charCodeAt(0);

    if (charCode === 13) {
      // Enter
      term.write("\r\n");
      if (loginState === "login_user") {
        username = inputBuffer;
        if (username.trim() === "") {
          promptLogin();
          return;
        }
        term.write(`Password: `);
        loginState = "login_pass";
        inputBuffer = "";
      } else if (loginState === "login_pass") {
        password = inputBuffer;
        term.write("Connecting...\r\n");
        loginState = "connecting"; // momentary state
        onConnect(username, password, sessionId);
      }
      return;
    } else if (charCode === 127) {
      // Backspace
      if (inputBuffer.length > 0) {
        inputBuffer = inputBuffer.slice(0, -1);
        term.write("\b \b");
      }
      return;
    } else if (charCode < 32) {
      return; // Ignore control chars
    }

    inputBuffer += key;
    if (loginState === "login_user") {
      term.write(key);
    } else {
      // Don't echo password
    }
  }

  onMount(() => {
    term = new Terminal({
      cursorBlink: true,
      fontFamily: "'JetBrains Mono', 'Fira Code', Consolas, monospace",
      fontSize: fontSize,
      theme: terminalThemes[activeTheme],
    });
    fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(termDiv);
    fitAddon.fit();

    term.onData((e) => {
      if (connected) {
        WriteToTerminal(sessionId, e.toString());
      } else {
        // Handle local input for login
        handleInput(e);
      }
    });

    term.onResize((size) => {
      if (connected) {
        ResizeTerminal(sessionId, size.rows, size.cols);
      }
    });

    resizeListener = window.addEventListener("resize", () => {
      fitAddon.fit();
    });

    cleanupData = EventsOn("terminal-data-" + sessionId, (data) => {
      term.write(data);
    });

    cleanupDisconnect = EventsOn("disconnected-" + sessionId, () => {
      term.write("\r\n[Disconnected]\r\n");
    });

    // Initial fit
    setTimeout(() => {
      fitAddon.fit();
      if (connected) {
        ResizeTerminal(sessionId, term.rows, term.cols);
      } else {
        // Auto-start login if we have a sessionId and aren't connected
        promptLogin();
      }
    }, 100);
  });

  // Method to start the login flow
  export function startLogin() {
    if (term) promptLogin();
  }
  onDestroy(() => {
    if (resizeListener) window.removeEventListener("resize", resizeListener);
    if (cleanupData) cleanupData();
    if (cleanupDisconnect) cleanupDisconnect();
    if (term) term.dispose();
  });

  // Expose fit method if needed
  export function fit() {
    if (fitAddon) fitAddon.fit();
  }

  export function focus() {
    if (term) term.focus();
  }

  function updateTheme() {
    if (term) term.options.theme = terminalThemes[activeTheme];
  }

  function updateFontSize() {
    if (term) {
      term.options.fontSize = fontSize;
      setTimeout(() => fit(), 50);
    }
  }
</script>

<div class="terminal-view-host">
  <div class="term-toolbar border-b glass">
    <div class="tool">
      <label for="theme-sel">Theme:</label>
      <select id="theme-sel" bind:value={activeTheme} on:change={updateTheme}>
        <option value="genpilot">Genpilot Dark</option>
        <option value="putty">Classic PuTTY</option>
        <option value="dracula">Dracula</option>
        <option value="solarized">Solarized Dark</option>
      </select>
    </div>
    <div class="tool">
      <label for="fs-sel">Size:</label>
      <input
        id="fs-sel"
        type="number"
        bind:value={fontSize}
        on:change={updateFontSize}
        min="8"
        max="32"
      />
    </div>
  </div>
  <div class="terminal-container" bind:this={termDiv}></div>
</div>

<style>
  .terminal-view-host {
    display: flex;
    flex-direction: column;
    height: 100%;
    width: 100%;
    background: #09090b;
  }
  .term-toolbar {
    padding: 4px 12px;
    display: flex;
    gap: 15px;
    align-items: center;
    background: #18181b;
  }
  .tool {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .tool label {
    font-size: 0.75em;
    font-weight: 600;
    color: #71717a;
    text-transform: uppercase;
  }
  .tool select,
  .tool input {
    background: #27272a;
    border: 1px solid #3f3f46;
    color: #e4e4e7;
    font-size: 0.8em;
    padding: 2px 6px;
    border-radius: 4px;
    outline: none;
  }
  .tool input {
    width: 50px;
  }

  .terminal-container {
    flex: 1;
    overflow: hidden;
  }

  /* Hide xterm scrollbar if custom scrollbar is preferred or let it handle itself */
  :global(.xterm-viewport) {
    overflow-y: auto;
  }
</style>
