import type { EventType, WSMessage } from '../types';

type MessageHandler = (msg: WSMessage) => void;

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private token: string;
  private roomId: string;
  private handlers: Map<EventType, Set<MessageHandler>> = new Map();
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private shouldReconnect: boolean = true;
  private onStatusChange?: (status: 'CONNECTING' | 'CONNECTED' | 'DISCONNECTED') => void;

  constructor(token: string, roomId: string, onStatusChange?: (status: 'CONNECTING' | 'CONNECTED' | 'DISCONNECTED') => void) {
    this.token = token;
    this.roomId = roomId;
    this.onStatusChange = onStatusChange;
  }

  public connect() {
    this.shouldReconnect = true;
    this.onStatusChange?.('CONNECTING');

    let url = (import.meta.env.VITE_WS_URL as string);
    if (!url) {
      const host = window.location.origin.includes('5173') ? 'localhost:8080' : window.location.host;
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      url = `${protocol}//${host}/ws`;
    }
    const fullUrl = `${url}${url.includes('?') ? '&' : '?'}token=${encodeURIComponent(this.token)}&roomId=${encodeURIComponent(this.roomId)}`;

    this.ws = new WebSocket(fullUrl);

    this.ws.onopen = () => {
      this.onStatusChange?.('CONNECTED');
    };

    this.ws.onmessage = (event) => {
      try {
        const rawLines = (event.data as string).split('\n');
        for (const line of rawLines) {
          if (!line.trim()) continue;
          const msg = JSON.parse(line) as WSMessage;
          const listeners = this.handlers.get(msg.type);
          if (listeners) {
            listeners.forEach((handler) => handler(msg));
          }
        }
      } catch (err) {
        console.error('Failed to parse WebSocket message:', err);
      }
    };

    this.ws.onclose = () => {
      this.onStatusChange?.('DISCONNECTED');
      if (this.shouldReconnect) {
        this.reconnectTimer = setTimeout(() => this.connect(), 2500);
      }
    };

    this.ws.onerror = (err) => {
      console.warn('WebSocket encountered error:', err);
      this.ws?.close();
    };
  }

  public on(type: EventType, handler: MessageHandler) {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, new Set());
    }
    this.handlers.get(type)!.add(handler);
    return () => {
      this.handlers.get(type)?.delete(handler);
    };
  }

  public send(type: EventType, payload: unknown) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return;
    }
    const msg: WSMessage = {
      type,
      roomId: this.roomId,
      payload,
      timestamp: Date.now(),
    };
    this.ws.send(JSON.stringify(msg));
  }

  // Convenience methods
  public sendPlay(position: number, rate: number = 1.0) {
    this.send('VIDEO_PLAY', { position, rate });
  }

  public sendPause(position: number, rate: number = 1.0) {
    this.send('VIDEO_PAUSE', { position, rate });
  }

  public sendSeek(position: number) {
    this.send('VIDEO_SEEK', { position });
  }

  public sendRate(rate: number) {
    this.send('VIDEO_RATE', { rate });
  }

  public sendSyncRequest(clientPosition: number) {
    this.send('SYNC_REQUEST', {
      clientPosition,
      clientTimestamp: Date.now(),
    });
  }

  public sendChatMessage(message: string) {
    this.send('CHAT_MESSAGE', { message });
  }

  public sendTyping(isTyping: boolean) {
    this.send('TYPING', { isTyping });
  }

  public sendUpdateSettings(onlyHostCanControl: boolean) {
    this.send('UPDATE_SETTINGS', { onlyHostCanControl });
  }

  public sendAudioChangeRequest(trackIndex: number, trackLabel: string, fromUser: string) {
    this.send('AUDIO_CHANGE_REQUEST', { trackIndex, trackLabel, fromUser });
  }

  public disconnect() {
    this.shouldReconnect = false;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
    }
    this.ws?.close();
    this.handlers.clear();
  }
}
