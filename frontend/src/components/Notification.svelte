<script context="module">
    import { writable } from "svelte/store";

    // Global store for notifications
    const _notifications = writable([]);

    export const notify = {
        subscribe: _notifications.subscribe,
        info: (msg, timeout = 3000) => add(msg, "info", timeout),
        success: (msg, timeout = 3000) => add(msg, "success", timeout),
        error: (msg, timeout = 5000) => add(msg, "error", timeout),
    };

    function add(message, type, timeout) {
        const id = Math.random().toString(36).substr(2, 9);
        _notifications.update((n) => [...n, { id, message, type }]);
        if (timeout > 0) {
            setTimeout(() => {
                _notifications.update((n) => n.filter((i) => i.id !== id));
            }, timeout);
        }
    }
</script>

<script>
    import { fade, fly } from "svelte/transition";
    import { flip } from "svelte/animate";

    // We can access _notifications from the module context in the template
    // using the $_notifications syntax if it was exported or assigned to a local variable?
    // Actually, Svelte 3/4 handles store auto-subscription with $ prefix
    // for top-level imports or variables.
    // Since _notifications is defined in context="module", we need to make sure
    // it's accessible to the instance.
    // Assigning it to a local constant to ensure reactivity scope.
    const notifications = _notifications;
</script>

<div class="notifications-container">
    {#each $notifications as n (n.id)}
        <div
            class="toast {n.type}"
            in:fly={{ y: 20, duration: 300 }}
            out:fade={{ duration: 200 }}
            animate:flip={{ duration: 300 }}
            on:click={() => {
                notifications.update((current) =>
                    current.filter((i) => i.id !== n.id),
                );
            }}
        >
            <div class="message">{n.message}</div>
        </div>
    {/each}
</div>

<style>
    .notifications-container {
        position: fixed;
        bottom: 20px;
        right: 20px;
        z-index: 9999;
        display: flex;
        flex-direction: column;
        gap: 10px;
        pointer-events: none; /* Allow clicks through container */
    }

    .toast {
        pointer-events: auto;
        padding: 12px 20px;
        border-radius: 4px;
        color: white;
        font-size: 0.9rem;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
        cursor: pointer;
        min-width: 200px;
        max-width: 400px;
        display: flex;
        align-items: center;
        border-left: 4px solid transparent;
        background: #333;
    }

    .toast.info {
        background: #252526;
        border-left-color: #007acc;
    }

    .toast.success {
        background: #252526;
        border-left-color: #10b981;
    }

    .toast.error {
        background: #252526;
        border-left-color: #ef4444;
    }

    .message {
        flex: 1;
        word-break: break-word;
    }
</style>
