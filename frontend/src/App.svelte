<script>
    import TerminalView from "./components/TerminalView.svelte";
    import FileView from "./components/FileView.svelte";
    import QueueView from "./components/QueueView.svelte";
    import TunnelsView from "./components/TunnelsView.svelte";
    import Sidebar from "./components/Sidebar.svelte";
    import logo from "./assets/logo.png";
    import SessionManager from "./components/SessionManager.svelte";
    import { Connect, DisconnectSession } from "../wailsjs/go/main/App";
    import { EventsOn, BrowserOpenURL } from "../wailsjs/runtime/runtime";
    import { onMount } from "svelte";
    import Notification, { notify } from "./components/Notification.svelte";

    let activeTab = "terminal"; // Default to terminal if session exists, else 'session'

    // Connection state moved to SessionManager, but we track active sessions here
    let activeSessions = []; // Each: { id, name, host, port, user, connected, activeTab }
    let activeSessionId = null;
    let status = "Ready";

    async function handleConnect(event) {
        const { name, host, port, user } = event.detail;

        if (!host) {
            status = "Host required";
            return;
        }

        const sessionId = "sess_" + Date.now();
        const newSession = {
            id: sessionId,
            name: name || host,
            host: host,
            port: port,
            user: user,
            connected: false,
            activeTab: "terminal",
        };

        activeSessions = [...activeSessions, newSession];
        activeSessionId = sessionId;
        activeTab = "terminal";

        status = "Prompting for credentials...";
    }

    // Callback from TerminalView
    async function onTerminalConnect(username, password, sid) {
        const sess = activeSessions.find((s) => s.id === sid);
        if (!sess) return;

        try {
            status = "Connecting...";
            await Connect(
                sid,
                sess.name,
                sess.host,
                Number(sess.port),
                String(username),
                String(password),
            );
            status = "Connected";
            sess.connected = true;
            sess.user = username;
            activeSessions = [...activeSessions];
            activeTab = "terminal";

            notify.success("Connected to " + sess.host);
        } catch (e) {
            status = "Connection failed";
            notify.error("Connection failed: " + e);
            // Re-prompt in terminal
            const term = terminalComponents[sid];
            if (term) term.startLogin();
        }
    }

    let terminalComponents = {};
    let fileComponents = {};

    async function handleDisconnect(sid) {
        const id = sid || activeSessionId;
        if (!id) return;
        try {
            await DisconnectSession(id);
            activeSessions = activeSessions.filter((s) => s.id !== id);
            if (activeSessionId === id) {
                activeSessionId =
                    activeSessions.length > 0 ? activeSessions[0].id : null;
                if (!activeSessionId) activeTab = "session";
            }
            notify.success("Disconnected session");
        } catch (e) {
            notify.error("Disconnect failed: " + e);
        }
    }

    onMount(() => {
        if (activeSessions.length === 0) {
            activeTab = "session";
        }
        // Since we emit disconnected-ID, we might not need a global listener,
        // but let's keep it if backend still emits 'disconnected' for something
        EventsOn("disconnected", (msg) => {
            notify.info("System: " + msg);
        });
    });

    function switchTab(tab) {
        activeTab = tab;
        // Logic moved to reactive statement below to handle Sidebar clicks too
    }

    // Reactive statement to handle tab changes and terminal fit
    $: if (activeSessionId && terminalComponents[activeSessionId]) {
        if (activeTab === "terminal") {
            setTimeout(() => {
                const term = terminalComponents[activeSessionId];
                if (term) {
                    term.fit();
                    term.focus();
                }
            }, 50);
        }
    }
</script>

<main>
    <Notification />
    <Sidebar bind:activeTab />

    <div class="content">
        <!-- Tab Bar - Only show if we have sessions -->
        {#if activeSessions.length > 0}
            <div class="tab-bar glass border-b">
                <button
                    class="tab-btn"
                    class:active={activeTab === "session"}
                    on:click={() => (activeTab = "session")}
                >
                    <span class="icon">➕</span> New
                </button>
                {#each activeSessions as s}
                    <button
                        class="tab-btn session-tab"
                        class:active={activeSessionId === s.id &&
                            activeTab !== "session" &&
                            activeTab !== "about"}
                        on:click={() => {
                            activeSessionId = s.id;
                            if (
                                activeTab === "session" ||
                                activeTab === "about"
                            ) {
                                activeTab = s.activeTab || "terminal";
                            }
                        }}
                    >
                        <span class="icon">{s.connected ? "🟢" : "🟡"}</span>
                        {s.name}
                        <!-- svelte-ignore a11y-click-events-have-key-events -->
                        <span
                            class="close-tab"
                            role="button"
                            tabindex="0"
                            on:click|stopPropagation={() =>
                                handleDisconnect(s.id)}>×</span
                        >
                    </button>
                {/each}
            </div>
        {/if}

        <!-- Session Creation / Management View -->
        <div
            class="tab-content"
            style:display={activeTab === "session" ? "flex" : "none"}
        >
            <SessionManager {activeSessions} on:connect={handleConnect} />
        </div>

        <!-- Active Session Content -->
        {#if activeSessionId && activeTab !== "session" && activeTab !== "about"}
            {#each activeSessions as s (s.id)}
                <div
                    class="tab-content-session"
                    style:display={activeSessionId === s.id ? "flex" : "none"}
                >
                    <div class="session-view-container">
                        <div
                            class="view-pane"
                            style:display={activeTab === "terminal"
                                ? "block"
                                : "none"}
                        >
                            <TerminalView
                                bind:this={terminalComponents[s.id]}
                                connected={s.connected}
                                sessionId={s.id}
                                onConnect={onTerminalConnect}
                            />
                        </div>
                        {#if activeTab === "files"}
                            <div class="view-pane">
                                <FileView sessionId={s.id} />
                            </div>
                        {/if}
                        {#if activeTab === "queue"}
                            <div class="view-pane">
                                <QueueView sessionId={s.id} />
                            </div>
                        {/if}
                        {#if activeTab === "tunnels"}
                            <div class="view-pane">
                                <TunnelsView sessionId={s.id} />
                            </div>
                        {/if}
                    </div>
                </div>
            {/each}
        {:else if activeTab !== "session" && activeTab !== "about"}
            <div class="no-session-msg">
                <div class="msg-box glass">
                    <h3>No active session</h3>
                    <p>Please connect or select a session to use this tool.</p>
                    <button
                        class="btn-connect"
                        on:click={() => (activeTab = "session")}
                        >Open Connection View</button
                    >
                </div>
            </div>
        {/if}

        <!-- 1. Connection View is handled by Display style lines 248-251 -->

        <div
            class="tab-content"
            style:display={activeTab === "about" ? "flex" : "none"}
        >
            <div class="about-msg">
                <img src={logo} alt="Genpilot Logo" class="about-logo" />
                <h1>Genpilot</h1>
                <p class="version">v4.4.0 (Wails Edition)</p>
                <p class="powered-by">Powered by Go + Svelte + xterm.js</p>

                <div class="author-block glass border">
                    <p class="author-name">
                        Author: <strong>faruk-guler</strong>
                    </p>
                    <div class="author-links">
                        <button
                            class="link-btn"
                            on:click={() =>
                                BrowserOpenURL(
                                    "https://github.com/faruk-guler",
                                )}
                            title="GitHub Profile"
                        >
                            <svg
                                xmlns="http://www.w3.org/2000/svg"
                                width="18"
                                height="18"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                ><path
                                    d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"
                                ></path></svg
                            >
                            <span>github.com/faruk-guler</span>
                            <svg
                                class="external-icon"
                                xmlns="http://www.w3.org/2000/svg"
                                width="14"
                                height="14"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                ><path
                                    d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"
                                ></path><polyline points="15 3 21 3 21 9"
                                ></polyline><line x1="10" y1="14" x2="21" y2="3"
                                ></line></svg
                            >
                        </button>
                        <button
                            class="link-btn"
                            on:click={() =>
                                BrowserOpenURL("https://www.farukguler.com")}
                            title="Personal Website"
                        >
                            <svg
                                xmlns="http://www.w3.org/2000/svg"
                                width="18"
                                height="18"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                ><circle cx="12" cy="12" r="10"></circle><line
                                    x1="2"
                                    y1="12"
                                    x2="22"
                                    y2="12"
                                ></line><path
                                    d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"
                                ></path></svg
                            >
                            <span>www.farukguler.com</span>
                            <svg
                                class="external-icon"
                                xmlns="http://www.w3.org/2000/svg"
                                width="14"
                                height="14"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                ><path
                                    d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"
                                ></path><polyline points="15 3 21 3 21 9"
                                ></polyline><line x1="10" y1="14" x2="21" y2="3"
                                ></line></svg
                            >
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </div>
</main>

<style>
    main {
        display: flex !important;
        flex-direction: column !important; /* Navbar top, Content bottom */
        height: 100vh;
        width: 100vw;
        background: var(--color-panel);
        color: var(--color-text);
        overflow: hidden;
    }

    .content {
        flex: 1;
        display: flex;
        flex-direction: column;
        overflow: hidden;
        background: var(--color-bg);
    }
    .tab-bar {
        display: flex;
        background: var(--color-panel);
        padding: 4px 8px 0 8px;
        gap: 4px;
    }
    .tab-btn {
        background: var(--color-surface);
        border: 1px solid var(--color-border);
        border-bottom: none;
        padding: 6px 15px;
        font-size: 0.8em;
        border-radius: 4px 4px 0 0;
        display: flex;
        align-items: center;
        gap: 8px;
        color: var(--color-muted);
        cursor: pointer;
        transition: all 0.2s;
        max-width: 200px;
        overflow: hidden;
        white-space: nowrap;
    }
    .tab-btn:hover {
        background: var(--color-border);
        color: var(--color-text);
    }
    .tab-btn.active {
        background: var(--color-bg);
        color: var(--color-text);
        border-color: var(--color-border);
        border-bottom: 2px solid var(--color-accent);
    }
    .close-tab {
        margin-left: 8px;
        opacity: 0.5;
        font-size: 1.2em;
    }
    .close-tab:hover {
        opacity: 1;
        color: #ef4444;
    }

    .tab-content-session {
        flex: 1;
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }
    .session-view-container {
        flex: 1;
        position: relative;
        overflow: hidden;
    }
    .view-pane {
        height: 100%;
        width: 100%;
    }
    .tab-content {
        flex: 1;
        display: flex;
        flex-direction: column;
        height: 100%;
        overflow: hidden;
    }

    button {
        padding: 6px 12px;
        border: none;
        cursor: pointer;
        border-radius: 3px;
        color: white;
    }

    .btn-connect {
        background: #007acc;
        font-weight: bold;
    }
    .btn-connect:hover {
        background: #0063a5;
    }

    .btn-connect:hover {
        background: #0063a5;
    }

    .about-msg {
        padding: 40px;
        text-align: center;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 100%;
    }

    .about-logo {
        width: 100px;
        height: 100px;
        margin-bottom: 20px;
        border-radius: var(--radius-lg);
        object-fit: contain;
    }

    .about-msg h1 {
        margin-bottom: 5px;
        font-size: 2.5em;
        color: var(--color-accent);
    }

    .about-msg .version {
        font-weight: bold;
        color: var(--color-text);
        margin-top: 0;
    }

    .about-msg .powered-by {
        color: var(--color-muted);
        margin-bottom: 30px;
    }

    .author-block {
        margin-top: 20px;
        padding: 24px;
        border-radius: var(--radius-lg);
        background: var(--color-surface);
        border: 1px solid var(--color-border);
        min-width: 300px;
    }

    .author-name {
        font-size: 1.1em;
        color: var(--color-text);
        margin-top: 0;
        margin-bottom: 16px;
    }

    .author-name strong {
        color: var(--color-accent);
    }

    .author-links {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .author-links .link-btn {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        color: var(--color-muted);
        background: none;
        border: none;
        cursor: pointer;
        padding: 6px 12px;
        transition: all 0.2s;
        font-size: 0.95em;
        font-family: inherit;
        border-radius: var(--radius-md);
    }

    .author-links .link-btn:hover {
        color: var(--color-text);
        background: rgba(255, 255, 255, 0.05);
    }

    .external-icon {
        opacity: 0.5;
        margin-left: 4px;
    }

    .link-btn:hover .external-icon {
        opacity: 1;
    }

    .no-session-msg {
        flex: 1;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 40px;
    }
    .msg-box {
        max-width: 400px;
        text-align: center;
        padding: 30px;
        border-radius: 12px;
        border: 1px solid var(--color-border);
    }
    .msg-box h3 {
        margin-top: 0;
        color: var(--color-text);
    }
    .msg-box p {
        color: var(--color-muted);
        margin-bottom: 20px;
    }
</style>
