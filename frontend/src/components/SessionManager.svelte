<script>
    import { createEventDispatcher, onMount } from "svelte";
    import {
        SaveSession,
        LoadSessions,
        DeleteSession,
    } from "../../wailsjs/go/main/App";
    import { notify } from "./Notification.svelte";

    export let activeSessions = [];

    const dispatch = createEventDispatcher();

    let sessions = [];
    let selectedSessionName = "";
    let newSessionName = "";
    let currentGroup = "";

    let host = "";
    let port = 22;
    let user = "";
    let pass = "";

    let status = "Ready";

    async function loadSavedSessions() {
        try {
            sessions = await LoadSessions();
        } catch (e) {
            console.error(e);
        }
    }

    function selectSession() {
        if (selectedSessionName === "") {
            host = "";
            port = 22;
            user = "";
            pass = "";
            return;
        }
        const s = sessions.find((x) => x.name === selectedSessionName);
        if (s) {
            host = s.host;
            port = s.port;
            user = s.username;
            currentGroup = s.group || "";
            pass = ""; // Password usually not loaded for security
        }
    }

    async function handleSave() {
        if (!newSessionName) {
            status = "Enter a name to save";
            return;
        }
        try {
            await SaveSession(
                String(newSessionName),
                String(host),
                String(user),
                String(pass),
                String(currentGroup),
                Number(port),
            );
            status = "Saved " + newSessionName;
            newSessionName = "";
            await loadSavedSessions();
            selectedSessionName = newSessionName;
            notify.success("Saved " + newSessionName);
            status = "Ready";
        } catch (e) {
            notify.error("Error saving: " + e);
            status = "Error";
        }
    }

    async function handleDelete() {
        if (!selectedSessionName) return;
        try {
            await DeleteSession(selectedSessionName);
            status = "Deleted " + selectedSessionName;
            selectedSessionName = "";
            await loadSavedSessions();
            selectSession(); // clear inputs
            notify.success("Deleted " + selectedSessionName);
        } catch (e) {
            notify.error("Error deleting: " + e);
        }
    }

    function handleConnectClick() {
        if (!host) {
            status = "Host required";
            return;
        }
        dispatch("connect", {
            name: selectedSessionName || host,
            host,
            port,
            user,
        });
    }

    onMount(() => {
        loadSavedSessions();
    });

    $: groupedSessions = sessions.reduce((acc, s) => {
        const group = s.group || "Default";
        if (!acc[group]) acc[group] = [];
        acc[group].push(s);
        return acc;
    }, {});
</script>

<div class="session-controls-wrapper">
    <!-- 1. Host and Port at the top -->
    <div class="connection-group">
        <label for="host-input">Host Name (or IP address)</label>
        <div class="input-row">
            <input
                id="host-input"
                bind:value={host}
                placeholder="e.g. 192.168.1.1"
                class="flex-3"
            />
            <div class="port-input">
                <label class="sub-label" for="port-input">Port</label>
                <input
                    id="port-input"
                    bind:value={port}
                    type="number"
                    placeholder="22"
                />
            </div>
        </div>
    </div>

    <!-- 2. Saved Sessions Area -->
    <div class="sessions-group">
        <label for="session-name-input"
            >Load, save or delete a stored session</label
        >
        <div class="saved-sessions-grid">
            <!-- Left: List and Input -->
            <div class="session-list-area">
                <label class="sub-label" for="session-name-input"
                    >Saved Sessions</label
                >
                <div class="input-wrapper">
                    <input
                        id="session-name-input"
                        bind:value={newSessionName}
                        placeholder="Session Name"
                        class="session-name-input"
                    />
                    <input
                        bind:value={currentGroup}
                        placeholder="Group (optional)"
                        class="group-name-input"
                    />
                </div>
                <div
                    class="session-list"
                    role="listbox"
                    aria-label="Saved Sessions List"
                    tabindex="0"
                >
                    {#each Object.entries(groupedSessions) as [group, groupSessions]}
                        <div class="session-group-label">{group}</div>
                        {#each groupSessions as s}
                            <!-- svelte-ignore a11y-click-events-have-key-events -->
                            <div
                                role="option"
                                aria-selected={selectedSessionName === s.name}
                                tabindex="0"
                                class="session-item indented"
                                class:selected={selectedSessionName === s.name}
                                on:click={() => {
                                    selectedSessionName = s.name;
                                    newSessionName = s.name;
                                    selectSession();
                                }}
                                on:keydown={(e) => {
                                    if (e.key === "Enter" || e.key === " ") {
                                        selectedSessionName = s.name;
                                        newSessionName = s.name;
                                        selectSession();
                                        e.preventDefault();
                                    }
                                }}
                                on:dblclick={() => {
                                    selectedSessionName = s.name;
                                    selectSession();
                                    handleConnectClick();
                                }}
                            >
                                {s.name}
                                {#if activeSessions.some((as) => as.name === s.name && as.connected)}
                                    <span class="active-dot" title="Connected"
                                    ></span>
                                {/if}
                            </div>
                        {/each}
                    {/each}
                </div>
            </div>

            <!-- Right: Action Buttons -->
            <div class="session-buttons">
                <button class="btn-action" on:click={selectSession}>Load</button
                >
                <button class="btn-action" on:click={handleSave}>Save</button>
                <button class="btn-action danger" on:click={handleDelete}
                    >Delete</button
                >
            </div>
        </div>
    </div>

    <div class="action-bar-bottom">
        <button class="btn-connect large" on:click={handleConnectClick}
            >Open</button
        >
    </div>

    <div class="status-bar-text">{status}</div>
</div>

<style>
    .session-controls-wrapper {
        background: var(--color-panel);
        padding: 20px;
        display: flex;
        flex-direction: column;
        gap: 15px;
        max-width: none;
        margin: 0;
        width: 100%;
        height: 100%;
        color: var(--color-text);
        box-sizing: border-box;
    }

    .connection-group {
        display: flex;
        flex-direction: column;
        gap: 5px;
    }
    .connection-group label {
        font-size: 0.9em;
        font-weight: bold;
    }
    .input-row {
        display: flex;
        gap: 10px;
        align-items: flex-end;
    }
    .flex-3 {
        flex: 3;
    }
    .port-input {
        display: flex;
        flex-direction: column;
        width: 80px;
    }
    .sub-label {
        font-size: 0.8em;
        margin-bottom: 2px;
    }

    .sessions-group {
        border-top: 1px solid var(--color-border);
        padding-top: 15px;
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .saved-sessions-grid {
        display: flex;
        gap: 20px;
        flex: 1;
        min-height: 0; /* Crucial for nested scrolling */
    }
    .session-list-area {
        flex: 3;
        display: flex;
        flex-direction: column;
        gap: 5px;
        height: 100%;
    }
    .input-wrapper {
        display: flex;
        flex-direction: column;
        gap: 5px;
    }
    .session-name-input {
        width: 100%;
    }
    .group-name-input {
        width: 100%;
        font-size: 0.85em;
        opacity: 0.8;
    }
    .session-list {
        background: var(--color-bg);
        border: 1px solid var(--color-border);
        flex: 1; /* Take remaining height */
        overflow-y: auto;
        padding: 2px;
    }
    .session-item {
        padding: 4px 8px;
        cursor: pointer;
        font-size: 0.9em;
        display: flex;
        justify-content: space-between;
        align-items: center;
    }
    .session-item.indented {
        padding-left: 20px;
    }
    .session-group-label {
        font-size: 0.8em;
        color: var(--color-muted);
        padding: 8px 8px 4px 8px;
        font-weight: bold;
        text-transform: uppercase;
        letter-spacing: 0.05em;
    }
    .session-item:hover {
        background: var(--color-surface);
    }
    .session-item.selected {
        background: var(--color-accent);
        color: white;
    }

    .session-buttons {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 8px;
        justify-content: flex-start;
        padding-top: 24px; /* Align with list */
    }
    .btn-action {
        background: var(--color-surface);
        border: 1px solid var(--color-border);
        padding: 6px 15px;
        text-align: center;
        color: var(--color-text);
        cursor: pointer;
    }
    .btn-action:hover {
        background: var(--color-border);
    }
    .btn-action.danger:hover {
        background: var(--color-danger);
    }

    .action-bar-bottom {
        display: flex;
        justify-content: flex-end;
        padding-top: 10px;
        border-top: 1px solid var(--color-border);
    }
    .btn-connect.large {
        padding: 8px 30px;
        font-size: 1.1em;
        background: var(--color-accent);
        color: white;
        border: none;
        cursor: pointer;
        border-radius: var(--radius-md);
    }
    .btn-connect:hover {
        background: var(--color-accent-hover);
    }

    .status-bar-text {
        font-size: 0.8em;
        color: var(--color-muted);
        text-align: right;
    }

    input {
        padding: 6px;
        background: var(--color-surface);
        border: 1px solid var(--color-border);
        color: var(--color-text);
        border-radius: var(--radius-sm);
        width: 100%;
        box-sizing: border-box;
    }

    .active-dot {
        display: inline-block;
        width: 8px;
        height: 8px;
        background-color: var(--color-success);
        border-radius: 50%;
        margin-left: 8px;
        box-shadow: 0 0 8px rgba(16, 185, 129, 0.4);
    }
</style>
