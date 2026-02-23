<script>
    import { onMount } from "svelte";
    import {
        StartLocalForward,
        StopLocalForward,
        GetActiveTunnels,
    } from "../../wailsjs/go/main/App";
    import { notify } from "./Notification.svelte";

    export let sessionId = "";

    let localPort = 8080;
    let remoteHost = "127.0.0.1";
    let remotePort = 80;
    
    let activeTunnels = [];
    let isWorking = false;

    async function refreshTunnels() {
        if (!sessionId) return;
        try {
            activeTunnels = (await GetActiveTunnels(sessionId)) || [];
        } catch (e) {
            console.error("Failed to fetch tunnels:", e);
        }
    }

    onMount(() => {
        refreshTunnels();
        const interval = setInterval(refreshTunnels, 2000);
        return () => clearInterval(interval);
    });

    async function handleStart() {
        if (!localPort || !remoteHost || !remotePort) {
            notify.error("Please fill in all tunnel details.");
            return;
        }

        isWorking = true;
        try {
            await StartLocalForward(
                sessionId,
                parseInt(localPort),
                remoteHost,
                parseInt(remotePort)
            );
            notify.success(`Tunnel created on local port ${localPort}`);
            localPort++; // Increment for convenience if they want to make another
            await refreshTunnels();
        } catch (e) {
            notify.error(`Tunnel failed: ${e}`);
        } finally {
            isWorking = false;
        }
    }

    async function handleStop(tunnelId) {
        try {
            await StopLocalForward(sessionId, tunnelId);
            notify.success(`Tunnel stopped`);
            await refreshTunnels();
        } catch (e) {
            notify.error(`Failed to stop tunnel: ${e}`);
        }
    }
</script>

<div class="tunnels-view">
    <!-- Header / Toolbar -->
    <div class="toolbar glass border-b">
        <h3>Local Port Forwarding</h3>
    </div>

    <!-- Content -->
    <div class="content-scroll">
        <div class="tunnel-form glass border">
            <h4>Create New Tunnel</h4>
            <div class="form-row">
                <div class="input-group">
                    <label for="local-port">Local Port</label>
                    <input id="local-port" type="number" bind:value={localPort} placeholder="e.g. 8080" />
                </div>
                <div class="divider-icon">➜</div>
                <div class="input-group">
                    <label for="remote-host">Remote Host</label>
                    <input id="remote-host" type="text" bind:value={remoteHost} placeholder="e.g. 127.0.0.1" />
                </div>
                <div class="divider-icon">:</div>
                <div class="input-group">
                    <label for="remote-port">Remote Port</label>
                    <input id="remote-port" type="number" bind:value={remotePort} placeholder="e.g. 80" />
                </div>
                
                <button class="btn-primary" on:click={handleStart} disabled={isWorking}>
                    {isWorking ? "Starting..." : "Start Tunnel"}
                </button>
            </div>
            <p class="help-text">Forwards connections to <code>localhost:LocalPort</code> -> <code>RemoteHost:RemotePort</code> via the SSH connection.</p>
        </div>

        <div class="active-tunnels">
            <h4>Active Tunnels</h4>
            
            {#if activeTunnels.length === 0}
                <div class="empty-state">
                    <p>No active tunnels for this session.</p>
                </div>
            {:else}
                <div class="tunnel-list">
                    {#each activeTunnels as t}
                        <div class="tunnel-item glass border">
                            <div class="tunnel-info">
                                <div class="tunnel-direction">
                                    <span class="local">localhost:{t.local_port}</span>
                                    <span class="arrow">➜</span>
                                    <span class="remote">{t.remote_host}:{t.remote_port}</span>
                                </div>
                            </div>
                            <div class="tunnel-actions">
                                <button class="btn-danger" on:click={() => handleStop(t.id)}>
                                    Stop
                                </button>
                            </div>
                        </div>
                    {/each}
                </div>
            {/if}
        </div>
    </div>
</div>

<style>
    .tunnels-view {
        display: flex;
        flex-direction: column;
        height: 100%;
        width: 100%;
        background: var(--color-bg);
        overflow: hidden;
    }

    .toolbar {
        height: 48px;
        display: flex;
        align-items: center;
        padding: 0 16px;
        background: var(--color-panel);
        flex-shrink: 0;
    }

    .toolbar h3 {
        margin: 0;
        font-size: 1.1em;
        font-weight: 600;
        color: var(--color-text);
    }

    .content-scroll {
        flex: 1;
        overflow-y: auto;
        padding: 24px;
        display: flex;
        flex-direction: column;
        gap: 24px;
    }

    .glass.border {
        background: var(--color-surface);
        border: 1px solid var(--color-border);
        border-radius: var(--radius-lg);
        padding: 20px;
    }

    h4 {
        margin-top: 0;
        margin-bottom: 16px;
        color: var(--color-text);
        font-weight: 600;
    }

    .form-row {
        display: flex;
        align-items: flex-end;
        gap: 16px;
        flex-wrap: wrap;
    }

    .input-group {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .input-group label {
        font-size: 0.8em;
        color: var(--color-muted);
        font-weight: 500;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .input-group input {
        background: var(--color-bg);
        border: 1px solid var(--color-border);
        color: var(--color-text);
        padding: 8px 12px;
        border-radius: 6px;
        font-size: 0.9em;
        outline: none;
        transition: border-color 0.2s;
    }
    
    .input-group input[type="number"] {
        width: 100px;
    }
    .input-group input[type="text"] {
        width: 180px;
    }

    .input-group input:focus {
        border-color: var(--color-accent);
    }

    .divider-icon {
        padding-bottom: 8px;
        color: var(--color-muted);
        font-weight: bold;
    }

    .btn-primary {
        background: var(--color-accent);
        color: #fff;
        border: none;
        padding: 8px 16px;
        border-radius: 6px;
        font-weight: 600;
        cursor: pointer;
        height: 35px; /* match input height */
        transition: opacity 0.2s;
        margin-bottom: 2px;
    }

    .btn-primary:hover:not(:disabled) {
        opacity: 0.9;
    }

    .btn-primary:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .help-text {
        margin-top: 16px;
        margin-bottom: 0;
        font-size: 0.85em;
        color: var(--color-muted);
    }

    .help-text code {
        background: var(--color-bg);
        padding: 2px 6px;
        border-radius: 4px;
        border: 1px solid var(--color-border);
    }

    .empty-state {
        padding: 30px;
        text-align: center;
        background: var(--color-surface);
        border: 1px dashed var(--color-border);
        border-radius: var(--radius-lg);
        color: var(--color-muted);
    }

    .tunnel-list {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .tunnel-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 16px 20px;
    }

    .tunnel-direction {
        display: flex;
        align-items: center;
        gap: 12px;
        font-family: monospace;
        font-size: 1.05em;
    }

    .local {
        color: var(--color-accent);
        font-weight: 600;
    }

    .arrow {
        color: var(--color-muted);
    }

    .remote {
        color: var(--color-text);
    }

    .btn-danger {
        background: transparent;
        border: 1px solid #ef4444;
        color: #ef4444;
        padding: 6px 12px;
        border-radius: 4px;
        font-size: 0.85em;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s;
    }

    .btn-danger:hover {
        background: #ef4444;
        color: #fff;
    }
</style>
