import { useState, useEffect, useRef } from 'react';
import type { User, Room, RoomMember, VideoState, Message, AudioChangePayload } from './types';
import { api, getStoredUser, clearAuthToken } from './services/api';
import { WebSocketClient } from './services/websocket';
import { ThemeProvider, useTheme } from './context/ThemeContext';
import { DyuetLogo } from './components/DyuetLogo';
import { AuthModal } from './auth/AuthModal';
import { RoomControls } from './room/RoomControls';
import { VideoPlayer } from './video/VideoPlayer';
import { ChatPanel } from './chat/ChatPanel';
import { PlusCircle, LogIn, Sun, Moon } from 'lucide-react';

function DyuetApp() {
  const { theme, toggleTheme } = useTheme();
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [room, setRoom] = useState<Room | null>(null);
  const [members, setMembers] = useState<RoomMember[]>([]);
  const [isHost, setIsHost] = useState<boolean>(false);
  const [joinCodeInput, setJoinCodeInput] = useState<string>('');
  const [wsStatus, setWsStatus] = useState<'CONNECTING' | 'CONNECTED' | 'DISCONNECTED'>('DISCONNECTED');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState<boolean>(false);

  // Chat sliding state & notification bubbles
  const [isChatOpen, setIsChatOpen] = useState<boolean>(true);
  const [unreadCount, setUnreadCount] = useState<number>(0);
  const [activeNotification, setActiveNotification] = useState<Message | null>(null);
  const [allMessages, setAllMessages] = useState<Message[]>([]);
  const notificationTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Audio Sync Suggestion Request state
  const [incomingAudioRequest, setIncomingAudioRequest] = useState<AudioChangePayload | null>(null);

  const [initialVideoState, setInitialVideoState] = useState<VideoState | undefined>(undefined);

  const wsClientRef = useRef<WebSocketClient | null>(null);
  const isDark = theme === 'dark';

  useEffect(() => {
    const saved = getStoredUser();
    if (saved) {
      setCurrentUser(saved);
    }
  }, []);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const joinCode = params.get('join');
    if (joinCode) {
      setJoinCodeInput(joinCode.toUpperCase());
    }
  }, []);

  useEffect(() => {
    if (!currentUser || !room) {
      if (wsClientRef.current) {
        wsClientRef.current.disconnect();
        wsClientRef.current = null;
      }
      return;
    }

    const token = localStorage.getItem('dyuet_token') || '';
    const client = new WebSocketClient(token, room.id, (status) => {
      setWsStatus(status);
    });

    client.on('ROOM_STATE', (msg) => {
      const payload = msg.payload as {
        room: Room;
        members: RoomMember[];
        messages: Message[];
        videoState: VideoState;
      };
      if (payload.room) setRoom(payload.room);
      if (payload.members) setMembers(payload.members);
      if (payload.messages) setAllMessages(payload.messages);
      if (payload.videoState) setInitialVideoState(payload.videoState);
    });

    client.on('USER_JOIN', (msg) => {
      const p = msg.payload as { userId: string; userName: string; avatar: string; isHost: boolean; isOnline: boolean };
      setMembers((prev) => {
        const exists = prev.find((m) => m.userId === p.userId);
        if (exists) {
          return prev.map((m) => (m.userId === p.userId ? { ...m, isOnline: true } : m));
        }
        return [
          ...prev,
          {
            roomId: room.id,
            userId: p.userId,
            user: { id: p.userId, name: p.userName, avatar: p.avatar, email: '' },
            isHost: p.isHost,
            isOnline: true,
            joinedAt: new Date().toISOString(),
          },
        ];
      });
    });

    client.on('USER_LEAVE', (msg) => {
      const p = msg.payload as { userId: string; isOnline: boolean };
      setMembers((prev) =>
        prev.map((m) => (m.userId === p.userId ? { ...m, isOnline: false } : m))
      );
    });

    client.on('UPDATE_SETTINGS', (msg) => {
      const p = msg.payload as { onlyHostCanControl: boolean };
      setRoom((prev) => (prev ? { ...prev, onlyHostCanControl: p.onlyHostCanControl } : null));
    });

    // Audio change recommendation request from another user
    client.on('AUDIO_CHANGE_REQUEST', (msg) => {
      const p = msg.payload as AudioChangePayload;
      setIncomingAudioRequest(p);
    });

    client.connect();
    wsClientRef.current = client;

    return () => {
      client.disconnect();
      wsClientRef.current = null;
    };
  }, [currentUser, room?.id]);

  const handleCreateRoom = async () => {
    if (!currentUser) return;
    setLoading(true);
    setError(null);
    try {
      const newRoom = await api.createRoom();
      setRoom(newRoom);
      setIsHost(true);
      setMembers([
        {
          roomId: newRoom.id,
          userId: currentUser.id,
          user: currentUser,
          isHost: true,
          isOnline: true,
          joinedAt: new Date().toISOString(),
        },
      ]);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create room');
    } finally {
      setLoading(false);
    }
  };

  const handleJoinRoom = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!currentUser || !joinCodeInput.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const res = await api.joinRoom(joinCodeInput.trim());
      setRoom(res.room);
      setIsHost(res.isHost);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Room not found or could not join');
    } finally {
      setLoading(false);
    }
  };

  const handleLeaveRoom = () => {
    if (wsClientRef.current) {
      wsClientRef.current.disconnect();
      wsClientRef.current = null;
    }
    setRoom(null);
    setMembers([]);
    setInitialVideoState(undefined);
    setUnreadCount(0);
    setActiveNotification(null);
    setIncomingAudioRequest(null);
    setAllMessages([]);
  };

  const handleLogout = () => {
    handleLeaveRoom();
    clearAuthToken();
    setCurrentUser(null);
  };

  const handleUpdateSettings = (onlyHostCanControl: boolean) => {
    wsClientRef.current?.sendUpdateSettings(onlyHostCanControl);
    setRoom((prev) => (prev ? { ...prev, onlyHostCanControl } : null));
  };

  // Handle incoming message to show floating notification bubble
  const handleNewMessage = (msg: Message) => {
    setAllMessages((prev) => [...prev, msg]);

    if (msg.userId === currentUser?.id || msg.isSystem) return;

    if (!isChatOpen) {
      setUnreadCount((c) => c + 1);
    }

    // Trigger floating chat bubble overlay
    setActiveNotification(msg);
    if (notificationTimeoutRef.current) {
      clearTimeout(notificationTimeoutRef.current);
    }
    notificationTimeoutRef.current = setTimeout(() => {
      setActiveNotification(null);
    }, 4500);
  };

  const handleToggleChat = () => {
    setIsChatOpen((prev) => {
      const next = !prev;
      if (next) {
        setUnreadCount(0);
        setActiveNotification(null);
      }
      return next;
    });
  };

  const handleOpenChat = () => {
    setIsChatOpen(true);
    setUnreadCount(0);
    setActiveNotification(null);
  };

  const handleSendChatMessage = (text: string) => {
    wsClientRef.current?.sendChatMessage(text);
  };

  if (!currentUser) {
    return <AuthModal onSuccess={(user) => setCurrentUser(user)} />;
  }

  // Refined Cinema Lobby Screen
  if (!room) {
    return (
      <div className="lobby-screen">
        {/* Top right theme toggle */}
        <div className="lobby-theme-toggle">
          <button
            type="button"
            onClick={toggleTheme}
            className="header-btn"
            title={isDark ? 'Switch to Light Mode' : 'Switch to Dark Mode'}
          >
            {isDark ? <Sun size={15} /> : <Moon size={15} />}
          </button>
        </div>

        {/* User Card */}
        <div className="lobby-user-bar">
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <img
              src={currentUser.avatar || `https://api.dicebear.com/7.x/initials/svg?seed=${encodeURIComponent(currentUser.name)}`}
              alt={currentUser.name}
              style={{ width: 34, height: 34, borderRadius: 'var(--radius-sm)' }}
            />
            <div>
              <h4 style={{ fontSize: 13, fontWeight: 700 }}>{currentUser.name}</h4>
              <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                {currentUser.email || 'Participant'}
              </span>
            </div>
          </div>

          <button
            type="button"
            onClick={handleLogout}
            className="header-btn"
            style={{ padding: '6px 12px', fontSize: 12 }}
          >
            Sign out
          </button>
        </div>

        {/* Lobby Box */}
        <div className="lobby-card">
          <div className="lobby-icon-badge">
            <DyuetLogo size={36} />
          </div>

          <h2 className="lobby-heading">DYUET</h2>
          <p className="lobby-subheading">
            Synchronized cinema session for any video
          </p>

          {error && <div className="auth-error" style={{ textAlign: 'center' }}>{error}</div>}

          {/* Action 1: Create Room */}
          <button
            type="button"
            disabled={loading}
            onClick={handleCreateRoom}
            className="btn-primary"
          >
            <PlusCircle size={16} />
            Create Watch Room
          </button>

          <div className="divider-row">
            <div className="divider-line" />
            <span className="divider-text">OR ENTER CODE</span>
            <div className="divider-line" />
          </div>

          {/* Action 2: Join Room */}
          <form onSubmit={handleJoinRoom}>
            <input
              type="text"
              maxLength={10}
              value={joinCodeInput}
              onChange={(e) => setJoinCodeInput(e.target.value.toUpperCase())}
              placeholder="ENTER ROOM CODE"
              className="form-input room-input"
            />

            <button
              type="submit"
              disabled={loading || !joinCodeInput.trim()}
              className="btn-secondary"
            >
              <LogIn size={15} />
              Join Session
            </button>
          </form>

          <div style={{ textAlign: 'center', fontSize: 11, color: 'var(--text-muted)', marginTop: 20 }}>
            Direct local playback • Zero video upload • High precision drift sync
          </div>
        </div>
      </div>
    );
  }

  // Active Theater View
  return (
    <div className="app-container">
      <RoomControls
        room={room}
        currentUser={currentUser}
        members={members}
        isHost={isHost}
        wsStatus={wsStatus}
        onUpdateSettings={handleUpdateSettings}
        onLeaveRoom={handleLeaveRoom}
        onLogout={handleLogout}
        isChatOpen={isChatOpen}
        onToggleChat={handleToggleChat}
        unreadCount={unreadCount}
      />

      <div className="theater-layout">
        <VideoPlayer
          room={room}
          isHost={isHost}
          currentUserName={currentUser.name}
          wsClient={wsClientRef.current}
          initialVideoState={initialVideoState}
          activeNotification={activeNotification}
          onDismissNotification={() => setActiveNotification(null)}
          onOpenChat={handleOpenChat}
          isChatOpen={isChatOpen}
          chatMessages={allMessages}
          onSendChatMessage={handleSendChatMessage}
          incomingAudioRequest={incomingAudioRequest}
          onDismissAudioRequest={() => setIncomingAudioRequest(null)}
        />

        <ChatPanel
          currentUser={currentUser}
          wsClient={wsClientRef.current}
          initialMessages={allMessages}
          isOpen={isChatOpen}
          onClose={() => setIsChatOpen(false)}
          onNewMessage={handleNewMessage}
        />
      </div>
    </div>
  );
}

export function App() {
  return (
    <ThemeProvider>
      <DyuetApp />
    </ThemeProvider>
  );
}

export default App;
