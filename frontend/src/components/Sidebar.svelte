<script>
    import { createEventDispatcher } from "svelte";

    export let activeTab = "session";
    const dispatch = createEventDispatcher();

    function switchTab(tab) {
        activeTab = tab;
        dispatch("change", tab);
    }
</script>

<div class="sidebar glass border-b">
    <div class="brand">
        <span class="brand-text">Genpilot</span>
    </div>

    <nav>
        <button
            class="nav-item no-drag"
            class:active={activeTab === "session"}
            on:click={() => switchTab("session")}
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
                ><rect x="2" y="3" width="20" height="14" rx="2" ry="2"
                ></rect><line x1="8" y1="21" x2="16" y2="21"></line><line
                    x1="12"
                    y1="17"
                    x2="12"
                    y2="21"
                ></line></svg
            >
            <span>Session</span>
        </button>

        <button
            class="nav-item no-drag"
            class:active={activeTab === "terminal"}
            on:click={() => switchTab("terminal")}
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
                ><polyline points="4 17 10 11 4 5"></polyline><line
                    x1="12"
                    y1="19"
                    x2="20"
                    y2="19"
                ></line></svg
            >
            <span>Terminal</span>
        </button>

        <button
            class="nav-item no-drag"
            class:active={activeTab === "files"}
            on:click={() => switchTab("files")}
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
                    d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"
                ></path><polyline points="14 2 14 8 20 8"></polyline><line
                    x1="16"
                    y1="13"
                    x2="8"
                    y2="13"
                ></line><line x1="16" y1="17" x2="8" y2="17"></line><polyline
                    points="10 9 9 9 8 9"
                ></polyline></svg
            >
            <span>Files</span>
        </button>

        <button
            class="nav-item no-drag"
            class:active={activeTab === "queue"}
            on:click={() => switchTab("queue")}
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
                ><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"
                ></polyline></svg
            >
            <span>Queue</span>
        </button>

        <button
            class="nav-item no-drag"
            class:active={activeTab === "tunnels"}
            on:click={() => switchTab("tunnels")}
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
                    d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"
                ></path><path
                    d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"
                ></path></svg
            >
            <span>Tunnels</span>
        </button>

        <button
            class="nav-item no-drag"
            class:active={activeTab === "about"}
            on:click={() => switchTab("about")}
            title="About / Settings"
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
                    x1="12"
                    y1="16"
                    x2="12"
                    y2="12"
                ></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg
            >
            <span>About</span>
        </button>
    </nav>
</div>

<style>
    .sidebar {
        width: 100%;
        height: 54px;
        display: flex;
        flex-direction: row;
        align-items: center;
        background-color: var(--color-panel);
        padding: 0 var(--spacing-lg);
        z-index: 9999;
        flex-shrink: 0;
        gap: var(--spacing-xl);
        pointer-events: auto !important;
        user-select: none;

        /* Remove global drag to prevent blocking clicks */
        --wails-draggable: none;
    }

    .no-drag,
    nav,
    .nav-item {
        /* Important: Make interactive elements non-draggable */
        --wails-draggable: no-drag !important;
        -webkit-app-region: no-drag !important;
    }

    .brand {
        display: flex;
        flex-direction: row;
        align-items: center;
        gap: 0.75rem;
        cursor: default;
    }

    .brand-text {
        font-weight: 600;
        font-size: 1rem;
        letter-spacing: -0.01em;
        display: block;
        color: var(--color-text);
    }

    nav {
        display: flex;
        flex-direction: row;
        gap: 12px;
        flex: 1; /* Push items or take space */
    }

    .nav-item {
        width: 44px;
        height: 44px;
        display: flex;
        flex-direction: column;
        justify-content: center;
        align-items: center;
        gap: 4px;
        border-radius: var(--radius-lg);
        color: var(--color-muted);
        transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
        cursor: pointer;
        position: relative;
    }
    .nav-item span {
        font-size: 0.65rem;
        font-weight: 500;
    }

    .nav-item:hover {
        background-color: var(--color-surface);
        color: var(--color-text);
    }

    .nav-item.active {
        background-color: var(--color-surface-hover);
        color: var(--color-accent);
        font-weight: 600;
    }

    .nav-item.active::after {
        content: "";
        position: absolute;
        bottom: -10px; /* Position indicator */
        left: 50%;
        transform: translateX(-50%);
        width: 16px;
        height: 3px;
        background-color: var(--color-accent);
        border-radius: 2px 2px 0 0;
        opacity: 0; /* Hidden for sidebar style, maybe useful for bottom nav? keeping simple */
        display: none;
    }
</style>
