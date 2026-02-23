<script>
    import { createEventDispatcher } from "svelte";
    const dispatch = createEventDispatcher();

    export let files = [];
    export let currentPath = ".";
    export let title = "Path";
    export let error = "";
    export let isRemote = false;

    let viewMode = "list"; // 'list' or 'grid'
    let selectedFiles = new Set();
    let lastSelectedIndex = -1;

    function formatSize(bytes) {
        const b = parseInt(bytes);
        if (isNaN(b) || b === 0) return "0 B";
        const k = 1024;
        const sizes = ["B", "KB", "MB", "GB", "TB"];
        const i = Math.floor(Math.log(b) / Math.log(k));
        return parseFloat((b / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
    }

    function handleDblClick(file) {
        if (file.is_dir) {
            dispatch("changePath", file.name);
        }
    }

    function toggleSelection(index, event) {
        const file = files[index];
        if (event.ctrlKey || event.metaKey) {
            if (selectedFiles.has(file.name)) {
                selectedFiles.delete(file.name);
            } else {
                selectedFiles.add(file.name);
            }
        } else if (event.shiftKey && lastSelectedIndex !== -1) {
            const start = Math.min(lastSelectedIndex, index);
            const end = Math.max(lastSelectedIndex, index);
            selectedFiles.clear();
            for (let i = start; i <= end; i++) {
                selectedFiles.add(files[i].name);
            }
        } else {
            selectedFiles.clear();
            selectedFiles.add(file.name);
        }
        selectedFiles = selectedFiles; // Trigger reactivity
        lastSelectedIndex = index;
        dispatch("select", Array.from(selectedFiles));
    }

    function handleKeydown(e, index) {
        if (e.key === "Enter") {
            handleDblClick(files[index]);
        } else if (e.key === " ") {
            toggleSelection(index, e);
            e.preventDefault();
        }
    }
</script>

<div class="file-pane border">
    <div class="pane-header glass border-b">
        <div class="pane-title">
            <span class="icon">{isRemote ? "🌐" : "💻"}</span>
            {title}
        </div>
        <div class="breadcrumb">
            <button
                class="btn-icon"
                on:click={() => dispatch("goUp")}
                title="Go Up"
            >
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"><path d="m18 15-6-6-6 6" /></svg
                >
            </button>
            <input
                type="text"
                bind:value={currentPath}
                on:keydown={(e) =>
                    e.key === "Enter" && dispatch("setPath", currentPath)}
            />
            <button
                class="btn-icon"
                on:click={() => dispatch("refresh")}
                title="Refresh"
            >
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    ><path
                        d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8"
                    /><path d="M21 3v5h-5" /></svg
                >
            </button>
        </div>
    </div>

    <div class="pane-content" class:grid={viewMode === "grid"}>
        {#if error}
            <div class="error-msg">{error}</div>
        {:else if files.length === 0}
            <div class="empty-msg">No files found</div>
        {:else}
            <div class="file-list" role="listbox">
                {#each files as file, i}
                    <!-- svelte-ignore a11y-click-events-have-key-events -->
                    <div
                        class="file-item"
                        class:selected={selectedFiles.has(file.name)}
                        class:is-dir={file.is_dir}
                        on:click={(e) => toggleSelection(i, e)}
                        on:dblclick={() => handleDblClick(file)}
                        on:keydown={(e) => handleKeydown(e, i)}
                        tabindex="0"
                        role="option"
                        aria-selected={selectedFiles.has(file.name)}
                    >
                        <span class="file-icon">
                            {#if file.is_dir}
                                <svg
                                    xmlns="http://www.w3.org/2000/svg"
                                    width="18"
                                    height="18"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="#eab308"
                                    stroke-width="2"
                                    stroke-linecap="round"
                                    stroke-linejoin="round"
                                    ><path
                                        d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2Z"
                                    /></svg
                                >
                            {:else}
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
                                        d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"
                                    /><polyline points="14 2 14 8 20 8" /></svg
                                >
                            {/if}
                        </span>
                        <span class="file-name">{file.name}</span>
                        <span class="file-size"
                            >{file.is_dir ? "--" : formatSize(file.size)}</span
                        >
                        <span class="file-time">{file.time}</span>
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</div>

<style>
    .file-pane {
        display: flex;
        flex-direction: column;
        height: 100%;
        background: #1e1e1e;
        overflow: hidden;
    }

    .pane-header {
        padding: 8px;
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .pane-title {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 0.85em;
        font-weight: 600;
        color: #999;
        text-transform: uppercase;
        letter-spacing: 0.05em;
    }

    .breadcrumb {
        display: flex;
        gap: 4px;
        align-items: center;
    }

    .breadcrumb input {
        flex: 1;
        background: #252526;
        border: 1px solid #3e3e42;
        color: #ccc;
        padding: 4px 8px;
        font-size: 0.9em;
        border-radius: 4px;
    }

    .pane-content {
        flex: 1;
        overflow-y: auto;
    }

    .file-list {
        display: flex;
        flex-direction: column;
    }

    .file-item {
        display: grid;
        grid-template-columns: 32px 1fr 100px 140px;
        align-items: center;
        padding: 4px 8px;
        cursor: default;
        font-size: 0.9em;
        border-bottom: 1px solid #2a2d2e;
        user-select: none;
    }

    .file-item:hover {
        background: #2a2d2e;
    }

    .file-item.selected {
        background: #094771;
        color: white;
    }

    .file-icon {
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .file-name {
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .file-size,
    .file-time {
        color: #888;
        font-size: 0.85em;
        text-align: right;
    }

    .file-item.selected .file-size,
    .file-item.selected .file-time {
        color: #ccc;
    }

    .btn-icon {
        background: transparent;
        border: none;
        color: #888;
        padding: 4px;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        border-radius: 4px;
    }

    .btn-icon:hover {
        background: #3e3e42;
        color: white;
    }

    .error-msg,
    .empty-msg {
        padding: 20px;
        text-align: center;
        color: #666;
    }

    .error-msg {
        color: #f87171;
    }
</style>
