import React, { useState } from 'react';
import type { Room, RoomMember, User } from '../types';
import { useTheme } from '../context/ThemeContext';
import { DyuetLogo } from '../components/DyuetLogo';
import {
  Copy,
  Check,
  Users,
  Shield,
  Crown,
  Sun,
  Moon,
  MessageSquare,
} from 'lucide-react';

interface RoomControlsProps {
  room: Room;
  currentUser: User;
  members: RoomMember[];
  isHost: boolean;
  wsStatus: 'CONNECTING' | 'CONNECTED' | 'DISCONNECTED';
  onUpdateSettings: (onlyHostCanControl: boolean) => void;
  onLeaveRoom: () => void;
  isChatOpen: boolean;
  onToggleChat: () => void;
  unreadCount: number;
}

export const RoomControls: React.FC<RoomControlsProps> = ({
  room,
  currentUser,
  members,
  isHost,
  wsStatus,
  onUpdateSettings,
  onLeaveRoom,
  isChatOpen,
  onToggleChat,
  unreadCount,
}) => {
  const { theme, toggleTheme } = useTheme();
  const [copied, setCopied] = useState(false);
  const [showMembers, setShowMembers] = useState(false);

  const isDark = theme === 'dark';
  const inviteUrl = `${window.location.origin}?join=${room.code}`;

  const handleCopy = () => {
    navigator.clipboard.writeText(inviteUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <header className="header-bar">
      {/* Left: Brand & Room Code */}
      <div className="header-left">
        <DyuetLogo size={28} />
        <span className="brand-title">Duet</span>

        {/* Room Code Badge */}
        <div className="room-code-badge">
          <span className="room-code-label">Room</span>
          <span className="room-code-value">{room.code}</span>
          <button
            type="button"
            onClick={handleCopy}
            title="Copy Invite Link"
            className="copy-btn"
          >
            {copied ? <Check size={14} style={{ color: 'var(--status-online)' }} /> : <Copy size={14} />}
          </button>
        </div>

        {/* Live Status Pill */}
        <div className="status-pill">
          <span
            className={`status-dot ${
              wsStatus === 'CONNECTED'
                ? 'online'
                : wsStatus === 'CONNECTING'
                ? 'connecting'
                : 'offline'
            }`}
          />
          <span>
            {wsStatus === 'CONNECTED'
              ? 'LIVE'
              : wsStatus === 'CONNECTING'
              ? 'CONNECTING'
              : 'OFFLINE'}
          </span>
        </div>
      </div>

      {/* Center: Permissions Switch */}
      <div className="header-center">
        <div className="control-switch">
          <Shield size={13} style={{ marginLeft: 6, color: 'var(--text-muted)' }} />
          <span className="control-label">Control:</span>
          <button
            type="button"
            disabled={!isHost}
            onClick={() => isHost && onUpdateSettings(false)}
            className={`control-pill ${!room.onlyHostCanControl ? 'active' : ''}`}
          >
            Everyone
          </button>
          <button
            type="button"
            disabled={!isHost}
            onClick={() => isHost && onUpdateSettings(true)}
            className={`control-pill ${room.onlyHostCanControl ? 'active' : ''}`}
          >
            Host Only
          </button>
        </div>
      </div>

      {/* Right: Theme Switcher, Chat Toggle, Members, User */}
      <div className="header-right">
        {/* Chat Slide Toggle Button */}
        <button
          type="button"
          onClick={onToggleChat}
          className={`header-btn ${isChatOpen ? 'active' : ''}`}
          title={isChatOpen ? 'Close Chat Drawer' : 'Open Chat Drawer'}
        >
          <MessageSquare size={15} />
          <span>Chat</span>
          {!isChatOpen && unreadCount > 0 && (
            <span className="unread-badge">{unreadCount}</span>
          )}
        </button>

        {/* Theme Toggle Button */}
        <button
          type="button"
          onClick={toggleTheme}
          className="header-btn"
          title={isDark ? 'Switch to Light Mode' : 'Switch to Dark Mode'}
        >
          {isDark ? <Sun size={15} /> : <Moon size={15} />}
        </button>

        {/* Members Button & Dropdown */}
        <div style={{ position: 'relative' }}>
          <button
            type="button"
            onClick={() => setShowMembers(!showMembers)}
            className="header-btn"
          >
            <Users size={15} />
            <span>{members.filter((m) => m.isOnline).length}</span>
          </button>

          {showMembers && (
            <div className="members-dropdown">
              <div className="dropdown-title">
                Active Participants ({members.length})
              </div>
              <div style={{ maxHeight: 240, overflowY: 'auto' }}>
                {members.map((member) => (
                  <div key={member.userId} className="member-item">
                    <div className="member-info">
                      <img
                        src={member.user?.avatar || `https://api.dicebear.com/7.x/initials/svg?seed=${encodeURIComponent(member.user?.name || 'User')}`}
                        alt={member.user?.name}
                        className="member-avatar"
                      />
                      <span className="member-name">
                        {member.user?.name || 'Member'}{' '}
                        {member.userId === currentUser.id ? '(You)' : ''}
                      </span>
                    </div>
                    {member.isHost && (
                      <span className="host-tag">
                        <Crown size={10} /> Host
                      </span>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* User profile & actions */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, paddingLeft: 8, borderLeft: '1px solid var(--border-subtle)' }}>
          <img
            src={currentUser.avatar || `https://api.dicebear.com/7.x/initials/svg?seed=${encodeURIComponent(currentUser.name)}`}
            alt={currentUser.name}
            style={{ width: 28, height: 28, borderRadius: 'var(--radius-sm)' }}
          />
          <span style={{ fontSize: 13, fontWeight: 700 }}>{currentUser.name}</span>
          <button
            type="button"
            onClick={onLeaveRoom}
            className="header-btn"
            style={{ padding: '6px 12px', fontSize: 12, fontWeight: 600 }}
          >
            Leave Room
          </button>
        </div>
      </div>
    </header>
  );
};
