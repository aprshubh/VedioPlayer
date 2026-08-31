import React, { useRef, useState, useEffect, useCallback } from 'react';
import type { VideoState, SyncCorrection, Room, Message, AudioChangePayload } from '../types';
import { WebSocketClient } from '../services/websocket';
import {
  Play,
  Pause,
  Volume2,
  VolumeX,
  Maximize,
  Minimize,
  Upload,
  RotateCcw,
  RotateCw,
  Film,
  Lock,
  MessageSquare,
  Settings,
  X,
  Send,
  Volume1,
  Check,
} from 'lucide-react';

interface VideoPlayerProps {
  room: Room;
  isHost: boolean;
  currentUserName: string;
  wsClient: WebSocketClient | null;
  initialVideoState?: VideoState;
  activeNotification: Message | null;
  onDismissNotification: () => void;
  onOpenChat: () => void;
  isChatOpen: boolean;
  chatMessages: Message[];
  onSendChatMessage: (msg: string) => void;
  incomingAudioRequest: AudioChangePayload | null;
  onDismissAudioRequest: () => void;
}

export const VideoPlayer: React.FC<VideoPlayerProps> = ({
  room,
  isHost,
  currentUserName,
  wsClient,
  initialVideoState,
  activeNotification,
  onDismissNotification,
  onOpenChat,
  isChatOpen,
  chatMessages,
  onSendChatMessage,
  incomingAudioRequest,
  onDismissAudioRequest,
}) => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const fsChatEndRef = useRef<HTMLDivElement>(null);

  const [videoSrc, setVideoSrc] = useState<string | null>(null);
  const [fileName, setFileName] = useState<string | null>(null);
  const [isPlaying, setIsPlaying] = useState<boolean>(false);
  const [currentTime, setCurrentTime] = useState<number>(0);
  const [duration, setDuration] = useState<number>(0);
  const [volume, setVolume] = useState<number>(1);
  const [isMuted, setIsMuted] = useState<boolean>(false);
  const [playbackRate, setPlaybackRate] = useState<number>(1.0);
  const [isFullscreen, setIsFullscreen] = useState<boolean>(false);
  const [showControls, setShowControls] = useState<boolean>(true);
  const [isDraggingOver, setIsDraggingOver] = useState<boolean>(false);

  // Fullscreen interactive chat overlay open state
  const [isFullscreenChatOpen, setIsFullscreenChatOpen] = useState<boolean>(false);
  const [fsInputMessage, setFsInputMessage] = useState<string>('');

  // Settings menu (Audio tracks)
  const [showSettings, setShowSettings] = useState<boolean>(false);
  const [selectedAudioTrack, setSelectedAudioTrack] = useState<number>(0);
  const [availableAudioTracks, setAvailableAudioTracks] = useState<string[]>([
    'Track 1 - Default Audio',
    'Track 2 - Alternate Language',
    'Track 3 - Surround / Commentary',
  ]);

  const [syncStatus, setSyncStatus] = useState<'SYNCED' | 'CATCHING_UP' | 'SEEKING' | 'WAITING_VIDEO'>('WAITING_VIDEO');
  const [driftMs, setDriftMs] = useState<number>(0);

  const isApplyingServerState = useRef<boolean>(false);
  const controlsTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const canControl = !room.onlyHostCanControl || isHost;

  const formatTime = (secs: number): string => {
    if (isNaN(secs) || secs < 0) return '00:00';
    const h = Math.floor(secs / 3600);
    const m = Math.floor((secs % 3600) / 60);
    const s = Math.floor(secs % 60);
    if (h > 0) {
      return `${h}:${m < 10 ? '0' : ''}${m}:${s < 10 ? '0' : ''}${s}`;
    }
    return `${m < 10 ? '0' : ''}${m}:${s < 10 ? '0' : ''}${s}`;
  };

  const handleFileSelect = (file: File) => {
    if (!file) return;
    if (videoSrc) {
      URL.revokeObjectURL(videoSrc);
    }
    const url = URL.createObjectURL(file);
    setVideoSrc(url);
    setFileName(file.name);
    setSyncStatus('SYNCED');

    // Inspect native audio tracks if browser supports audioTracks API
    setTimeout(() => {
      const video = videoRef.current as any;
      if (video && video.audioTracks && video.audioTracks.length > 0) {
        const detected: string[] = [];
        for (let i = 0; i < video.audioTracks.length; i++) {
          const t = video.audioTracks[i];
          detected.push(t.label || t.language || `Audio Track ${i + 1}`);
        }
        setAvailableAudioTracks(detected);
      }
    }, 500);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDraggingOver(true);
  };
  const handleDragLeave = () => setIsDraggingOver(false);
  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDraggingOver(false);
    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      handleFileSelect(e.dataTransfer.files[0]);
    }
  };

  useEffect(() => {
    const handleFullscreenChange = () => {
      const isFs = !!document.fullscreenElement;
      setIsFullscreen(isFs);
      if (!isFs) {
        setIsFullscreenChatOpen(false);
      }
    };
    document.addEventListener('fullscreenchange', handleFullscreenChange);
    return () => document.removeEventListener('fullscreenchange', handleFullscreenChange);
  }, []);

  // Auto-scroll fullscreen chat to latest message
  useEffect(() => {
    if (isFullscreenChatOpen) {
      fsChatEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [chatMessages, isFullscreenChatOpen]);

  // WebSocket Event Listeners for Playback Synchronization
  useEffect(() => {
    if (!wsClient) return;

    const unsubPlay = wsClient.on('VIDEO_PLAY', (msg) => {
      const payload = msg.payload as VideoState;
      const video = videoRef.current;
      if (!video) return;

      isApplyingServerState.current = true;
      if (Math.abs(video.currentTime - payload.position) > 0.3) {
        video.currentTime = payload.position;
      }
      video.playbackRate = payload.rate || 1.0;
      video.play().catch(() => {});
      setIsPlaying(true);
      setPlaybackRate(payload.rate || 1.0);
      setSyncStatus('SYNCED');
      setTimeout(() => {
        isApplyingServerState.current = false;
      }, 300);
    });

    const unsubPause = wsClient.on('VIDEO_PAUSE', (msg) => {
      const payload = msg.payload as VideoState;
      const video = videoRef.current;
      if (!video) return;

      isApplyingServerState.current = true;
      video.pause();
      if (Math.abs(video.currentTime - payload.position) > 0.2) {
        video.currentTime = payload.position;
      }
      setIsPlaying(false);
      setSyncStatus('SYNCED');
      setTimeout(() => {
        isApplyingServerState.current = false;
      }, 300);
    });

    const unsubSeek = wsClient.on('VIDEO_SEEK', (msg) => {
      const payload = msg.payload as VideoState;
      const video = videoRef.current;
      if (!video) return;

      isApplyingServerState.current = true;
      video.currentTime = payload.position;
      setCurrentTime(payload.position);
      setSyncStatus('SEEKING');
      setTimeout(() => {
        isApplyingServerState.current = false;
        setSyncStatus('SYNCED');
      }, 300);
    });

    const unsubRate = wsClient.on('VIDEO_RATE', (msg) => {
      const payload = msg.payload as VideoState;
      const video = videoRef.current;
      if (!video) return;

      video.playbackRate = payload.rate;
      setPlaybackRate(payload.rate);
    });

    const unsubCorrection = wsClient.on('SYNC_CORRECTION', (msg) => {
      const correction = msg.payload as SyncCorrection;
      const video = videoRef.current;
      if (!video || !videoSrc) return;

      const drift = correction.drift;
      setDriftMs(Math.round(drift * 1000));

      if (correction.action === 'HARD_SEEK') {
        isApplyingServerState.current = true;
        video.currentTime = correction.serverPosition;
        setSyncStatus('SEEKING');
        setTimeout(() => {
          isApplyingServerState.current = false;
          setSyncStatus('SYNCED');
        }, 300);
      } else if (correction.action === 'RATE_ADJUST') {
        video.playbackRate = correction.targetRate;
        setSyncStatus('CATCHING_UP');
      } else {
        if (video.playbackRate !== correction.rate) {
          video.playbackRate = correction.rate;
        }
        setSyncStatus('SYNCED');
      }

      if (correction.playing && video.paused) {
        video.play().catch(() => {});
        setIsPlaying(true);
      } else if (!correction.playing && !video.paused) {
        video.pause();
        setIsPlaying(false);
      }
    });

    return () => {
      unsubPlay();
      unsubPause();
      unsubSeek();
      unsubRate();
      unsubCorrection();
    };
  }, [wsClient, videoSrc]);

  // Periodic drift check
  useEffect(() => {
    if (!wsClient || !videoSrc) return;
    const interval = setInterval(() => {
      const video = videoRef.current;
      if (video && !video.seeking) {
        wsClient.sendSyncRequest(video.currentTime);
      }
    }, 3000);
    return () => clearInterval(interval);
  }, [wsClient, videoSrc]);

  // Initial video state
  useEffect(() => {
    if (!initialVideoState || !videoRef.current) return;
    const video = videoRef.current;
    if (initialVideoState.position > 0) {
      video.currentTime = initialVideoState.position;
    }
    if (initialVideoState.playing) {
      video.play().catch(() => {});
      setIsPlaying(true);
    }
  }, [initialVideoState]);

  const togglePlay = useCallback(() => {
    if (!canControl) return;
    const video = videoRef.current;
    if (!video) return;

    if (video.paused) {
      video.play().catch(() => {});
      setIsPlaying(true);
      wsClient?.sendPlay(video.currentTime, playbackRate);
    } else {
      video.pause();
      setIsPlaying(false);
      wsClient?.sendPause(video.currentTime, playbackRate);
    }
  }, [canControl, playbackRate, wsClient]);

  const handleSeek = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (!canControl) return;
    const target = parseFloat(e.target.value);
    const video = videoRef.current;
    if (!video) return;

    video.currentTime = target;
    setCurrentTime(target);
    wsClient?.sendSeek(target);
  };

  // Dedicated -10s and +10s forward/backward steps
  const handleStep = (seconds: number) => {
    if (!canControl || !videoRef.current) return;
    const newTime = Math.max(0, Math.min(duration || Infinity, videoRef.current.currentTime + seconds));
    videoRef.current.currentTime = newTime;
    setCurrentTime(newTime);
    wsClient?.sendSeek(newTime);
  };

  const handleRateChange = (newRate: number) => {
    if (!canControl) return;
    const video = videoRef.current;
    if (!video) return;

    video.playbackRate = newRate;
    setPlaybackRate(newRate);
    wsClient?.sendRate(newRate);
  };

  // Audio track change and sync request recommendation
  const handleSelectAudioTrack = (index: number) => {
    setSelectedAudioTrack(index);
    const video = videoRef.current as any;
    if (video && video.audioTracks && video.audioTracks.length > index) {
      for (let i = 0; i < video.audioTracks.length; i++) {
        video.audioTracks[i].enabled = (i === index);
      }
    }
    setShowSettings(false);

    // Send audio sync request to other room members
    const label = availableAudioTracks[index] || `Track ${index + 1}`;
    wsClient?.sendAudioChangeRequest(index, label, currentUserName);
  };

  const handleAcceptIncomingAudio = () => {
    if (!incomingAudioRequest) return;
    const targetIndex = incomingAudioRequest.trackIndex;
    setSelectedAudioTrack(targetIndex);
    const video = videoRef.current as any;
    if (video && video.audioTracks && video.audioTracks.length > targetIndex) {
      for (let i = 0; i < video.audioTracks.length; i++) {
        video.audioTracks[i].enabled = (i === targetIndex);
      }
    }
    onDismissAudioRequest();
  };

  const toggleMute = () => {
    const video = videoRef.current;
    if (!video) return;
    video.muted = !video.muted;
    setIsMuted(video.muted);
  };

  const handleVolumeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newVol = parseFloat(e.target.value);
    const video = videoRef.current;
    if (!video) return;
    video.volume = newVol;
    setVolume(newVol);
    if (newVol > 0 && isMuted) {
      video.muted = false;
      setIsMuted(false);
    }
  };

  const toggleFullscreen = () => {
    if (!containerRef.current) return;
    if (!document.fullscreenElement) {
      containerRef.current.requestFullscreen().catch(() => {});
    } else {
      document.exitFullscreen().catch(() => {});
    }
  };

  const handleMouseMove = () => {
    setShowControls(true);
    if (controlsTimeoutRef.current) {
      clearTimeout(controlsTimeoutRef.current);
    }
    if (isPlaying) {
      controlsTimeoutRef.current = setTimeout(() => {
        setShowControls(false);
      }, 3000);
    }
  };

  const handleSendFsChat = (e: React.FormEvent) => {
    e.preventDefault();
    if (!fsInputMessage.trim()) return;
    onSendChatMessage(fsInputMessage.trim());
    setFsInputMessage('');
  };

  return (
    <div
      ref={containerRef}
      onMouseMove={handleMouseMove}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
      className="video-player-container"
    >
      {/* Empty State / File Selector Prompt */}
      {!videoSrc ? (
        <div className="empty-player-prompt">
          <div className="empty-icon-card">
            <Film size={32} />
          </div>

          <h3 className="empty-title">Select Movie File</h3>

          <p className="empty-desc">
            Drag & drop any video file (<code>.mp4</code>, <code>.mkv</code>, <code>.webm</code>) or browse from your computer. Plays locally with zero buffering.
          </p>

          <label className="file-select-btn">
            <Upload size={16} />
            Choose Video File
            <input
              type="file"
              accept="video/*,.mkv"
              onChange={(e) => e.target.files?.[0] && handleFileSelect(e.target.files[0])}
              style={{ display: 'none' }}
            />
          </label>
        </div>
      ) : (
        <video
          ref={videoRef}
          src={videoSrc}
          onClick={togglePlay}
          onTimeUpdate={() => videoRef.current && setCurrentTime(videoRef.current.currentTime)}
          onDurationChange={() => videoRef.current && setDuration(videoRef.current.duration)}
          onPlay={() => setIsPlaying(true)}
          onPause={() => setIsPlaying(false)}
          className="video-element"
        />
      )}

      {/* Drag & Drop Visual Target */}
      {isDraggingOver && (
        <div className="drag-overlay">
          <div className="drag-content">
            <Upload size={40} style={{ margin: '0 auto 8px auto' }} />
            <h4>Drop Video File Here</h4>
          </div>
        </div>
      )}

      {/* Sync Health HUD (Top Left) */}
      {videoSrc && (
        <div className="sync-hud">
          <span
            className={`sync-dot ${
              syncStatus === 'SYNCED' ? 'synced' : 'adjusting'
            }`}
          />
          <span className="sync-text">
            {syncStatus === 'SYNCED' ? 'SYNCED' : 'ADJUSTING'}
          </span>
          {driftMs !== 0 && (
            <span className="sync-drift">
              {driftMs > 0 ? `+${driftMs}` : driftMs}ms
            </span>
          )}
          {fileName && (
            <span className="video-filename" title={fileName}>
              {fileName}
            </span>
          )}
        </div>
      )}

      {/* Permission Lock indicator */}
      {!canControl && videoSrc && (
        <div className="perm-lock">
          <Lock size={13} />
          <span>Host Only Controls</span>
        </div>
      )}

      {/* ========================================================== */}
      {/* AUDIO CHANGE REQUEST NOTIFICATION BANNER                   */}
      {/* ========================================================== */}
      {incomingAudioRequest && (
        <div className="audio-request-banner">
          <Volume1 size={18} style={{ color: 'var(--status-warn)', flexShrink: 0 }} />
          <div className="audio-request-text">
            <strong>{incomingAudioRequest.fromUser}</strong> requested to switch audio to{' '}
            <strong>{incomingAudioRequest.trackLabel}</strong>. Switch too?
          </div>
          <div className="audio-request-actions">
            <button
              type="button"
              onClick={handleAcceptIncomingAudio}
              className="btn-accept-audio"
            >
              Switch Audio
            </button>
            <button
              type="button"
              onClick={onDismissAudioRequest}
              className="btn-dismiss-audio"
            >
              Dismiss
            </button>
          </div>
        </div>
      )}

      {/* ========================================================== */}
      {/* FLOATING CHAT BUBBLE NOTIFICATION OVERLAY                  */}
      {/* ========================================================== */}
      {activeNotification && (!isChatOpen || isFullscreen) && !isFullscreenChatOpen && (
        <div className="floating-chat-bubble-container">
          <div
            className="floating-chat-toast"
            onClick={() => {
              if (isFullscreen) {
                setIsFullscreenChatOpen(true);
              } else {
                onOpenChat();
              }
              onDismissNotification();
            }}
            title="Click to open chat"
          >
            <img
              src={activeNotification.avatar || `https://api.dicebear.com/7.x/initials/svg?seed=${encodeURIComponent(activeNotification.userName)}`}
              alt={activeNotification.userName}
              className="toast-avatar"
            />
            <div className="toast-content">
              <div className="toast-header">
                <span className="toast-name">{activeNotification.userName}</span>
                <span className="toast-badge">New Message</span>
              </div>
              <div className="toast-message">{activeNotification.message}</div>
            </div>
          </div>
        </div>
      )}

      {/* ========================================================== */}
      {/* FULLSCREEN TRANSPARENT FLOATING CHAT OVERLAY               */}
      {/* ========================================================== */}
      {isFullscreen && isFullscreenChatOpen && (
        <div className="fullscreen-chat-panel">
          <button
            type="button"
            onClick={() => setIsFullscreenChatOpen(false)}
            className="fs-floating-close-btn"
            title="Close Chat"
          >
            <X size={14} />
          </button>

          <div className="fs-chat-messages">
            {chatMessages.filter((m) => !m.isSystem).length === 0 ? (
              <div style={{ textAlign: 'right', color: 'rgba(255, 255, 255, 0.45)', fontSize: 11, padding: 8 }}>
                No messages yet
              </div>
            ) : (
              chatMessages
                .filter((m) => !m.isSystem)
                .map((msg) => {
                  const isMe = msg.userName === currentUserName;
                  return (
                    <div
                      key={msg.id}
                      className={`fs-msg-item ${isMe ? 'is-me' : 'is-other'}`}
                    >
                      {!isMe && (
                        <span className="fs-msg-author">
                          {msg.userName}
                        </span>
                      )}
                      <div className="fs-msg-bubble">
                        {msg.message}
                      </div>
                    </div>
                  );
                })
            )}
            <div ref={fsChatEndRef} />
          </div>

          <form onSubmit={handleSendFsChat} className="fs-chat-input-area">
            <div className="fs-input-pill">
              <input
                type="text"
                value={fsInputMessage}
                onChange={(e) => setFsInputMessage(e.target.value)}
                placeholder="Type message..."
                autoFocus
              />
              <button
                type="submit"
                disabled={!fsInputMessage.trim()}
                className="fs-input-send-btn"
              >
                <Send size={12} />
              </button>
            </div>
          </form>
        </div>
      )}

      {/* ========================================================== */}
      {/* SETTINGS MENU (AUDIO TRACKS / DELAY)                       */}
      {/* ========================================================== */}
      {showSettings && videoSrc && (
        <div className="video-settings-menu">
          <div className="settings-header">
            <span className="settings-title">Audio & Video Settings</span>
            <button
              type="button"
              onClick={() => setShowSettings(false)}
              className="ctrl-btn"
              style={{ padding: 4 }}
            >
              <X size={14} />
            </button>
          </div>

          <div className="settings-section-label">Select Audio Track</div>
          <div style={{ marginBottom: 12 }}>
            {availableAudioTracks.map((label, idx) => (
              <button
                key={label}
                type="button"
                onClick={() => handleSelectAudioTrack(idx)}
                className={`audio-track-item ${selectedAudioTrack === idx ? 'selected' : ''}`}
              >
                <span>{label}</span>
                {selectedAudioTrack === idx && <Check size={14} />}
              </button>
            ))}
          </div>

          <div style={{ fontSize: 10, color: 'var(--text-muted)', lineHeight: 1.4 }}>
            💡 Changing audio suggests this track to other participants so everyone stays in sync.
          </div>
        </div>
      )}

      {/* Cinema Controls Bar (Bottom) */}
      {videoSrc && (
        <div className={`cinema-controls ${!showControls && isPlaying ? 'hidden' : ''}`}>
          {/* Timeline scrub bar */}
          <div className="timeline-row">
            <input
              type="range"
              min={0}
              max={duration || 100}
              step={0.1}
              value={currentTime}
              disabled={!canControl}
              onChange={handleSeek}
            />
          </div>

          {/* Controls row */}
          <div className="controls-row">
            {/* Left group */}
            <div className="controls-group">
              <button
                type="button"
                disabled={!canControl}
                onClick={togglePlay}
                className="ctrl-btn"
                title={isPlaying ? 'Pause (Space)' : 'Play (Space)'}
              >
                {isPlaying ? <Pause size={18} /> : <Play size={18} fill="#ffffff" />}
              </button>

              {/* Backward -10s */}
              <button
                type="button"
                disabled={!canControl}
                onClick={() => handleStep(-10)}
                className="ctrl-btn"
                title="Rewind 10 seconds"
              >
                <RotateCcw size={15} />
                <span className="step-badge">10</span>
              </button>

              {/* Forward +10s */}
              <button
                type="button"
                disabled={!canControl}
                onClick={() => handleStep(10)}
                className="ctrl-btn"
                title="Forward 10 seconds"
              >
                <RotateCw size={15} />
                <span className="step-badge">10</span>
              </button>

              <div className="timestamp-display">
                <span className="current">{formatTime(currentTime)}</span> / {formatTime(duration)}
              </div>

              {/* Volume */}
              <div className="volume-wrapper">
                <button
                  type="button"
                  onClick={toggleMute}
                  className="ctrl-btn"
                >
                  {isMuted || volume === 0 ? <VolumeX size={15} /> : <Volume2 size={15} />}
                </button>
                <input
                  type="range"
                  min={0}
                  max={1}
                  step={0.05}
                  value={isMuted ? 0 : volume}
                  onChange={handleVolumeChange}
                />
              </div>
            </div>

            {/* Right group */}
            <div className="controls-group">
              {/* Fullscreen Chat Overlay Button */}
              {isFullscreen && (
                <button
                  type="button"
                  onClick={() => setIsFullscreenChatOpen(!isFullscreenChatOpen)}
                  className={`ctrl-btn ${isFullscreenChatOpen ? 'active' : ''}`}
                  title="Toggle Fullscreen Chat Panel"
                >
                  <MessageSquare size={16} />
                </button>
              )}

              {/* Playback speed selector */}
              <select
                disabled={!canControl}
                value={playbackRate}
                onChange={(e) => handleRateChange(parseFloat(e.target.value))}
                className="speed-select"
              >
                <option value={0.5}>0.5x</option>
                <option value={0.75}>0.75x</option>
                <option value={1.0}>1.0x</option>
                <option value={1.25}>1.25x</option>
                <option value={1.5}>1.5x</option>
                <option value={2.0}>2.0x</option>
              </select>

              {/* Audio & Settings button */}
              <button
                type="button"
                onClick={() => setShowSettings(!showSettings)}
                className={`ctrl-btn ${showSettings ? 'active' : ''}`}
                title="Audio Track Settings"
              >
                <Settings size={16} />
              </button>

              {/* Change video file */}
              <label className="ctrl-btn" title="Change video">
                <Upload size={15} />
                <input
                  type="file"
                  accept="video/*,.mkv"
                  onChange={(e) => e.target.files?.[0] && handleFileSelect(e.target.files[0])}
                  style={{ display: 'none' }}
                />
              </label>

              {/* Fullscreen */}
              <button
                type="button"
                onClick={toggleFullscreen}
                className="ctrl-btn"
                title={isFullscreen ? 'Exit Fullscreen' : 'Fullscreen'}
              >
                {isFullscreen ? <Minimize size={16} /> : <Maximize size={16} />}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
