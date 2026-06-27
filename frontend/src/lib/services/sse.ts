/**
 * sse.ts — SSE EventSource client with automatic reconnection and
 * Last-Event-ID tracking.
 */

/** Parsed SSE event delivered to the application. */
export interface SSEEvent {
  id: number;
  type: string;
  data: unknown;
}

export type SSEEventHandler = (evt: SSEEvent) => void;

export interface SSEClientOptions {
  /**
   * Called before each reconnect attempt. Return false to stop reconnecting
   * (e.g. session deleted / 404).
   */
  beforeReconnect?: () => Promise<boolean>;
  /** Max reconnect attempts after errors. 0 = unlimited. Default: 60. */
  maxReconnectAttempts?: number;
  /** Delay between reconnect attempts in ms. Default: 2000. */
  reconnectDelayMs?: number;
}

export interface SSEClient {
  close: () => void;
}

const DEFAULT_MAX_RECONNECTS = 60;
const DEFAULT_RECONNECT_DELAY_MS = 2000;

export function createSSEClient(
  url: string,
  onEvent: SSEEventHandler,
  onError?: () => void,
  options?: SSEClientOptions,
): SSEClient {
  let es: EventSource | null = null;
  let lastEventID = 0;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  let closed = false;
  let reconnectAttempts = 0;

  const maxReconnects = options?.maxReconnectAttempts ?? DEFAULT_MAX_RECONNECTS;
  const reconnectDelayMs =
    options?.reconnectDelayMs ?? DEFAULT_RECONNECT_DELAY_MS;
  const beforeReconnect = options?.beforeReconnect;

  function buildURL(): string {
    if (lastEventID === 0) return url;
    const sep = url.includes("?") ? "&" : "?";
    return `${url}${sep}lastEventId=${lastEventID}`;
  }

  function scheduleReconnect(): void {
    if (closed) return;

    reconnectAttempts++;
    if (maxReconnects > 0 && reconnectAttempts > maxReconnects) {
      closed = true;
      return;
    }

    reconnectTimer = setTimeout(() => {
      void attemptReconnect();
    }, reconnectDelayMs);
  }

  async function attemptReconnect(): Promise<void> {
    reconnectTimer = null;
    if (closed) return;

    if (beforeReconnect) {
      try {
        const ok = await beforeReconnect();
        if (!ok) {
          closed = true;
          return;
        }
      } catch {
        // Transient probe failure — schedule another reconnect attempt.
        scheduleReconnect();
        return;
      }
    }

    connect();
  }

  function connect(): void {
    if (closed) return;

    es = new EventSource(buildURL());

    es.onmessage = (raw: MessageEvent) => {
      handleRaw(raw, "");
    };

    const knownTypes = [
      "iteration.start",
      "agent.started",
      "agent.complete",
      "agent.error",
      "agent.phase",
      "iteration.complete",
      "session.finalized",
      "doc.phase",
      "doc.token",
      "doc.complete",
      "agent.token",
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
      scheduleReconnect();
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
    reconnectAttempts = 0;
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

/** True when the URL targets GET /sessions/{id}/events. */
export function isSessionEventsURL(url: string): boolean {
  try {
    const path = new URL(url, "http://localhost").pathname;
    const parts = path.split("/").filter(Boolean);
    return (
      parts.length === 3 && parts[0] === "sessions" && parts[2] === "events"
    );
  } catch {
    return false;
  }
}
