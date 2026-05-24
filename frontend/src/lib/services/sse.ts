/**
 * sse.ts — SSE EventSource client with automatic reconnection and
 * Last-Event-ID tracking.
 *
 * Usage:
 *   const client = createSSEClient(url, (evt) => handleEvent(evt));
 *   // ...later:
 *   client.close();
 *
 * Design notes (§8.22 docs/PLAN.md):
 *   - Uses the native browser EventSource API (no WebSocket, no polling).
 *   - Tracks the last received event ID and passes it as `Last-Event-ID` on
 *     reconnect so the server can replay missed events from its ring buffer.
 *   - Auto-reconnects with a 2-second delay on any `onerror` event.
 *   - close() tears down the connection and cancels any pending reconnect.
 */

/** Parsed SSE event delivered to the application. */
export interface SSEEvent {
  /** Monotonically increasing per-session event counter from the server. */
  id: number;
  /** Event type string (e.g. "agent.started", "iteration.complete"). */
  type: string;
  /** JSON-decoded event payload. */
  data: unknown;
}

/** Callback invoked for every received SSEEvent. */
export type SSEEventHandler = (evt: SSEEvent) => void;

/** Returned handle — call close() to disconnect. */
export interface SSEClient {
  close: () => void;
}

/**
 * createSSEClient opens a server-sent events connection to `url` and invokes
 * `onEvent` for each received event.
 *
 * @param url     - Absolute or relative URL of the SSE endpoint.
 * @param onEvent - Called with each parsed SSEEvent.
 * @param onError - Optional callback invoked when the connection is lost
 *                  (before the automatic reconnect attempt).
 */
export function createSSEClient(
  url: string,
  onEvent: SSEEventHandler,
  onError?: () => void,
): SSEClient {
  let es: EventSource | null = null;
  let lastEventID = 0;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  let closed = false;

  function buildURL(): string {
    if (lastEventID === 0) return url;
    const sep = url.includes("?") ? "&" : "?";
    return `${url}${sep}lastEventId=${lastEventID}`;
  }

  function connect(): void {
    if (closed) return;

    // EventSource does not natively send Last-Event-ID as a query param —
    // the browser handles it via the `Last-Event-ID` header on reconnect, but
    // since we manage reconnects ourselves we encode it in the URL.
    es = new EventSource(buildURL());

    es.onmessage = (raw: MessageEvent) => {
      // Generic message handler (no event type set).
      handleRaw(raw, "");
    };

    // Attach named event listeners for the known event types.
    const knownTypes = [
      "iteration.start",
      "agent.started",
      "agent.complete",
      "agent.error",
      "iteration.complete",
      "session.finalized",
    ];
    for (const type of knownTypes) {
      es.addEventListener(type, (raw: Event) => {
        handleRaw(raw as MessageEvent, type);
      });
    }

    es.onerror = () => {
      if (closed) return;
      es?.close();
      es = null;
      onError?.();
      // Reconnect after 2 s.
      reconnectTimer = setTimeout(() => {
        reconnectTimer = null;
        connect();
      }, 2000);
    };
  }

  function handleRaw(raw: MessageEvent, fallbackType: string): void {
    const type = (raw as MessageEvent & { type?: string }).type || fallbackType;
    const idStr = (raw as MessageEvent & { lastEventId?: string }).lastEventId;
    const id = idStr ? parseInt(idStr, 10) : 0;
    if (!isNaN(id) && id > lastEventID) {
      lastEventID = id;
    }

    let data: unknown = raw.data;
    try {
      data = JSON.parse(raw.data as string);
    } catch {
      // Leave data as raw string if JSON parsing fails.
    }

    onEvent({ id, type, data });
  }

  connect();

  return {
    close(): void {
      closed = true;
      if (reconnectTimer !== null) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
      }
      es?.close();
      es = null;
    },
  };
}
