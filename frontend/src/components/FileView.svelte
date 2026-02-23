<script>
  import { onMount } from "svelte";
  import {
    ListFiles,
    GoUp,
    UploadFile,
    DownloadFile,
    SelectUploadFile,
    SelectSavePath,
  } from "../../wailsjs/go/main/App";
  import FilePane from "./FilePane.svelte";
  import { notify } from "./Notification.svelte";

  export let sessionId = "";

  // Remote State
  let remotePath = ".";
  let remoteFiles = [];
  let remoteError = "";

  async function loadRemoteFiles() {
    try {
      remoteFiles = await ListFiles(sessionId, remotePath);
      remoteError = "";
    } catch (e) {
      remoteError = e.toString();
      remoteFiles = [];
    }
  }

  async function goUpRemote() {
    remotePath = await GoUp(sessionId, remotePath);
    loadRemoteFiles();
  }

  function handlePathChangeRemote(event) {
    const dir = event.detail;
    if (remotePath.endsWith("/")) remotePath += dir;
    else remotePath += "/" + dir;
    loadRemoteFiles();
  }

  // Action: Push (Upload from Local to Remote)
  async function handlePush() {
    try {
      const localPath = await SelectUploadFile();
      if (!localPath) return; // User cancelled

      // Get filename from path (basic logic, improves with path lib if needed)
      const filename = localPath.split(/[\\/]/).pop();
      const targetRemotePath = remotePath.endsWith("/")
        ? remotePath + filename
        : remotePath + "/" + filename;

      await UploadFile(sessionId, localPath, targetRemotePath);
      notify.success(`Queued push: ${filename}`);
    } catch (e) {
      notify.error(`Push failed: ${e}`);
    }
  }

  // Action: Pull (Download from Remote to Local)
  async function handlePull() {
    if (selectedRemote.length === 0) return;

    for (const name of selectedRemote) {
      const sourceRemotePath = remotePath.endsWith("/")
        ? remotePath + name
        : remotePath + "/" + name;

      try {
        // For single file, ask where to save. For multiple, we might need folder selection?
        // Wails SaveFileDialog is for single file.
        // If multiple selected, we'd need a directory selector or loop dialogs (bad UX).
        // Let's assume single selection for Export for v1, or just loop for now.
        // Better UX: If multiple, maybe just download to Downloads folder?
        // For now, let's implement single file export flow clearly.

        let targetLocalPath = await SelectSavePath(name);
        if (!targetLocalPath) continue; // Skip or cancel

        await DownloadFile(sessionId, sourceRemotePath, targetLocalPath);
        notify.success(`Queued pull: ${name}`);
      } catch (e) {
        notify.error(`Pull failed: ${e}`);
      }
    }
  }

  onMount(() => {
    loadRemoteFiles();
  });

  let selectedRemote = [];
</script>

<div class="single-file-view">
  <!-- Toolbar -->
  <div class="toolbar glass border-b">
    <div class="toolbar-actions">
      <button
        class="btn-tool"
        on:click={handlePush}
        title="Push file to remote server"
      >
        <span class="icon">⬆️</span> Push
      </button>
      <button
        class="btn-tool"
        on:click={handlePull}
        disabled={selectedRemote.length === 0}
        title="Pull selected file(s) to local machine"
      >
        <span class="icon">⬇️</span> Pull
      </button>
      <span class="divider"></span>
      <button class="btn-tool" on:click={loadRemoteFiles} title="Refresh">
        <span class="icon">🔄</span> Refresh
      </button>
    </div>
  </div>

  <!-- Content -->
  <div class="pane-wrapper">
    <FilePane
      title="Remote Server"
      isRemote={true}
      currentPath={remotePath}
      files={remoteFiles}
      error={remoteError}
      on:changePath={handlePathChangeRemote}
      on:goUp={goUpRemote}
      on:refresh={loadRemoteFiles}
      on:setPath={(e) => {
        remotePath = e.detail;
        loadRemoteFiles();
      }}
      on:select={(e) => (selectedRemote = e.detail)}
    />
  </div>
</div>

<style>
  .single-file-view {
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

  .toolbar-actions {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .btn-tool {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 12px;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    color: var(--color-text);
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.85em;
    font-weight: 500;
    transition: all 0.2s;
  }

  .btn-tool:hover:not(:disabled) {
    background: var(--color-border);
  }

  .btn-tool:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .divider {
    width: 1px;
    height: 24px;
    background: var(--color-border);
    margin: 0 4px;
  }

  .pane-wrapper {
    flex: 1;
    overflow: hidden;
    padding: 0;
  }
</style>
