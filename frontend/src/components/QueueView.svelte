<script>
    import { onMount, onDestroy } from "svelte";
    import { fade, slide } from "svelte/transition";
    import { EventsOn, EventsOff } from "../../wailsjs/runtime/runtime";
    import {
        GetTransfers,
        CancelTransfer,
        ClearCompletedTransfers,
    } from "../../wailsjs/go/main/App";

    export let sessionId = "";

    let transfers = [];
    let cleanup = null;

    function updateTransfers(items) {
        if (!items) items = [];
        const now = new Date().getTime();

        // Sort items: Active first, then pending, then completed
        transfers = items
            .sort((a, b) => {
                return b.ID - a.ID; // Newest first
            })
            .map((item) => {
                // Calculate Speed and ETA
                if (item.Status === 1 && item.start_time) {
                    // InProgress
                    const start = new Date(item.start_time).getTime();
                    const duration = (now - start) / 1000; // seconds
                    if (duration > 0) {
                        item.speedBytes = item.transfer_bytes / duration;
                        if (item.speedBytes > 0) {
                            const remainingResponse =
                                item.total_bytes - item.transfer_bytes;
                            item.etaSeconds =
                                remainingResponse / item.speedBytes;
                        }
                    }
                }
                return item;
            });
    }

    async function loadTransfers() {
        try {
            const items = await GetTransfers(sessionId);
            updateTransfers(items);
        } catch (e) {
            console.error("Failed to load transfers", e);
        }
    }

    async function cancel(id) {
        await CancelTransfer(sessionId, id);
    }

    async function clearCompleted() {
        await ClearCompletedTransfers(sessionId);
        loadTransfers();
    }

    onMount(() => {
        loadTransfers();
        // Listen for backend events
        cleanup = EventsOn("transfer-update-" + sessionId, (data) => {
            updateTransfers(data);
        });
    });

    onDestroy(() => {
        if (cleanup) cleanup();
        // Fallback or explicit off if cleanup not returned (older Wails?)
        // But we know Wails v2 runtime returns it.
    });

    function getStatusClass(status) {
        // Status: 0=Pending, 1=InProgress, 2=Paused, 3=Completed, 4=Failed, 5=Cancelled
        switch (status) {
            case 0:
                return "status-pending";
            case 1:
                return "status-active";
            case 2:
                return "status-paused";
            case 3:
                return "status-success";
            case 4:
                return "status-error";
            case 5:
                return "status-muted";
            default:
                return "";
        }
    }

    function getStatusLabel(status) {
        switch (status) {
            case 0:
                return "Pending";
            case 1:
                return "Uploading...";
            case 2:
                return "Paused";
            case 3:
                return "Complete";
            case 4:
                return "Failed";
            case 5:
                return "Cancelled";
            default:
                return "Unknown";
        }
    }

    function formatBytes(bytes) {
        if (!bytes || bytes === 0) return "0 B";
        const k = 1024;
        const sizes = ["B", "KB", "MB", "GB", "TB"];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
    }

    function formatTime(seconds) {
        if (!seconds || !isFinite(seconds)) return "--";
        if (seconds < 60) return Math.round(seconds) + "s";
        const m = Math.floor(seconds / 60);
        if (m < 60) return m + "m " + Math.round(seconds % 60) + "s";
        return Math.floor(m / 60) + "h " + (m % 60) + "m";
    }
</script>

<div class="queue-container">
    <div class="queue-header">
        <h2>Transfer Queue</h2>
        <div class="header-actions">
            <span class="badge">{transfers.length} items</span>
            {#if transfers.length > 0}
                <button class="btn-text" on:click={clearCompleted}
                    >Clear Completed</button
                >
            {/if}
        </div>
    </div>

    <div class="queue-list glass">
        <div class="list-header">
            <span class="col-name">Name</span>
            <span class="col-type">Type</span>
            <span class="col-progress">Progress</span>
            <span class="col-status">Status</span>
            <span class="col-action"></span>
        </div>

        {#if transfers.length === 0}
            <div class="empty-state" in:fade>
                <div class="icon-circle">
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        width="32"
                        height="32"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        ><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"
                        ></path><polyline points="17 8 12 3 7 8"
                        ></polyline><line x1="12" y1="3" x2="12" y2="15"
                        ></line></svg
                    >
                </div>
                <h3>No active transfers</h3>
                <p>Upload or download files to see them here.</p>
            </div>
        {:else}
            <div class="transfer-items">
                {#each transfers as item (item.ID)}
                    <div class="transfer-row" transition:slide|local>
                        <div class="col-name">
                            <div class="file-icon">
                                {item.Direction === 1 ? "⬆️" : "⬇️"}
                            </div>
                            <div class="name-info">
                                <span class="filename" title={item.LocalPath}
                                    >{item.FileName}</span
                                >
                                {#if item.Status === 1}
                                    <span class="speed-info"
                                        >{formatBytes(item.TransferBytes)} of {formatBytes(
                                            item.TotalBytes,
                                        )}
                                        {#if item.speedBytes}
                                            • {formatBytes(item.speedBytes)}/s
                                        {/if}
                                        {#if item.etaSeconds}
                                            • ETA: {formatTime(item.etaSeconds)}
                                        {/if}
                                    </span>
                                {/if}
                            </div>
                        </div>
                        <div class="col-type">
                            <span class="type-badge"
                                >{item.Direction === 1
                                    ? "Upload"
                                    : "Download"}</span
                            >
                        </div>
                        <div class="col-progress">
                            {#if item.Status === 1}
                                <div class="progress-bar">
                                    <div
                                        class="progress-fill"
                                        style="width: {item.Progress()}%"
                                    ></div>
                                </div>
                            {:else}
                                <div class="progress-bar empty"></div>
                            {/if}
                        </div>
                        <div class="col-status">
                            <span
                                class="status-badge {getStatusClass(
                                    item.Status,
                                )}">{getStatusLabel(item.Status)}</span
                            >
                        </div>
                        <div class="col-action">
                            {#if item.Status === 1 || item.Status === 0}
                                <button
                                    class="btn-icon"
                                    on:click={() => cancel(item.ID)}
                                    title="Cancel"
                                >
                                    <svg
                                        xmlns="http://www.w3.org/2000/svg"
                                        width="14"
                                        height="14"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        stroke-width="2"
                                        stroke-linecap="round"
                                        stroke-linejoin="round"
                                        ><line x1="18" y1="6" x2="6" y2="18"
                                        ></line><line
                                            x1="6"
                                            y1="6"
                                            x2="18"
                                            y2="18"
                                        ></line></svg
                                    >
                                </button>
                            {/if}
                        </div>
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</div>

<style>
    .queue-container {
        display: flex;
        flex-direction: column;
        height: 100%;
        padding: var(--spacing-lg);
        gap: var(--spacing-md);
        background-color: var(--color-bg);
        color: var(--color-text);
        overflow: hidden;
    }

    .queue-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding-bottom: var(--spacing-sm);
        border-bottom: 1px solid var(--color-border);
    }

    .header-actions {
        display: flex;
        align-items: center;
        gap: 1rem;
    }

    .btn-text {
        background: none;
        border: none;
        color: var(--color-accent);
        cursor: pointer;
        font-size: 0.8rem;
    }
    .btn-text:hover {
        text-decoration: underline;
    }

    h2 {
        font-size: 1.25rem;
        font-weight: 600;
        margin: 0;
        letter-spacing: -0.01em;
    }

    .badge {
        background: var(--color-surface);
        padding: 2px 8px;
        border-radius: 99px;
        font-size: 0.75rem;
        color: var(--color-muted);
        border: 1px solid var(--color-border);
    }

    .queue-list {
        flex: 1;
        display: flex;
        flex-direction: column;
        border: 1px solid var(--color-border);
        border-radius: var(--radius-lg);
        overflow: hidden;
        background: var(--color-panel);
    }

    .list-header {
        display: flex;
        padding: var(--spacing-sm) var(--spacing-lg);
        background: var(--color-surface);
        border-bottom: 1px solid var(--color-border);
        font-size: 0.8rem;
        font-weight: 500;
        color: var(--color-muted);
        text-transform: uppercase;
        letter-spacing: 0.05em;
    }

    .col-name {
        flex: 3;
        display: flex;
        align-items: center;
        gap: 0.75rem;
        overflow: hidden;
    }
    .col-type {
        width: 100px;
        display: flex;
        align-items: center;
    }
    .col-progress {
        flex: 2;
        display: flex;
        align-items: center;
        padding-right: 1rem;
    }
    .col-status {
        width: 120px;
        display: flex;
        align-items: center;
        justify-content: flex-end;
    }
    .col-action {
        width: 40px;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .empty-state {
        flex: 1;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: var(--spacing-md);
        color: var(--color-muted);
        opacity: 0.8;
    }

    .icon-circle {
        width: 64px;
        height: 64px;
        border-radius: 50%;
        background: var(--color-surface);
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--color-accent);
        box-shadow: var(--shadow-sm);
        border: 1px solid var(--color-border);
    }

    h3 {
        font-size: 1rem;
        font-weight: 500;
        color: var(--color-text);
        margin: 0;
    }

    p {
        font-size: 0.9rem;
        margin: 0;
    }

    .transfer-items {
        flex: 1;
        overflow-y: auto;
    }

    .transfer-row {
        display: flex;
        padding: 0.75rem var(--spacing-lg);
        border-bottom: 1px solid var(--color-border);
        font-size: 0.9rem;
    }

    .filename {
        display: block;
        font-weight: 500;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        max-width: 100%;
    }

    .name-info {
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }

    .speed-info {
        font-size: 0.75rem;
        color: var(--color-muted);
    }

    .type-badge {
        font-size: 0.75rem;
        padding: 2px 6px;
        background: var(--color-surface);
        border-radius: 4px;
        color: var(--color-muted);
    }

    .progress-bar {
        width: 100%;
        height: 6px;
        background: var(--color-surface);
        border-radius: 3px;
        overflow: hidden;
    }

    .progress-fill {
        height: 100%;
        background: var(--color-accent);
        transition: width 0.3s ease;
    }

    .status-badge {
        font-size: 0.75rem;
        font-weight: 500;
    }

    .status-pending {
        color: var(--color-muted);
    }
    .status-active {
        color: var(--color-accent);
    }
    .status-success {
        color: #10b981;
    }
    .status-error {
        color: #ef4444;
    }
    .status-cancelled {
        color: var(--color-muted);
        text-decoration: line-through;
    }

    .btn-icon {
        background: none;
        border: none;
        color: var(--color-muted);
        cursor: pointer;
        padding: 4px;
        border-radius: 4px;
    }
    .btn-icon:hover {
        background: var(--color-surface);
        color: #ef4444;
    }
</style>
