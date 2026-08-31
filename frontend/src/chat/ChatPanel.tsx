import React, { useState, useEffect, useRef } from 'react';
import type { Message, User } from '../types';
import { WebSocketClient } from '../services/websocket';
import { Send, MessageSquare, ChevronRight } from 'lucide-react';

interface ChatPanelProps {
  currentUser: User;
  wsClient: WebSocketClient | null;
  initialMessages?: Message[];
  isOpen: boolean;
  onClose: () => void;
  onNewMessage?: (msg: Message) => void;
}

export const ChatPanel: React.FC<ChatPanelProps> = ({
  currentUser,
  wsClient,
  initialMessages = [],
  isOpen,
  onClose,
  onNewMessage,
}) => {
  const [messages, setMessages] = useState<Message[]>(initialMessages);
  const [inputMessage, setInputMessage] = useState('');
  const [typingUsers, setTypingUsers] = useState<string[]>([]);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const typingTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    if (isOpen) {
      scrollToBottom();
    }
  }, [messages, isOpen]);

  useEffect(() => {
    if (!wsClient) return;

    const unsubMsg = wsClient.on('CHAT_MESSAGE', (msg) => {
      const chatMsg = msg.payload as Message;
      setMessages((prev) => [...prev, chatMsg]);
      onNewMessage?.(chatMsg);
    });

    const unsubTyping = wsClient.on('TYPING', (msg) => {
      const payload = msg.payload as { isTyping: boolean; userName: string };
      if (payload.isTyping) {
        setTypingUsers((prev) => Array.from(new Set([...prev, payload.userName])));
      } else {
        setTypingUsers((prev) => prev.filter((name) => name !== payload.userName));
      }
    });

    return () => {
      unsubMsg();
      unsubTyping();
    };
  }, [wsClient, onNewMessage]);

  const handleSendMessage = (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputMessage.trim() || !wsClient) return;

    wsClient.sendChatMessage(inputMessage.trim());
    setInputMessage('');
    wsClient.sendTyping(false);
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setInputMessage(e.target.value);
    if (!wsClient) return;

    wsClient.sendTyping(true);
    if (typingTimeoutRef.current) {
      clearTimeout(typingTimeoutRef.current);
    }
    typingTimeoutRef.current = setTimeout(() => {
      wsClient.sendTyping(false);
    }, 2000);
  };

  const formatTime = (dateStr: string) => {
    try {
      const d = new Date(dateStr);
      return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    } catch {
      return '';
    }
  };

  return (
    <div className={`chat-drawer ${!isOpen ? 'collapsed' : ''}`}>
      {/* Header with slide/collapse toggle */}
      <div className="chat-header">
        <div className="chat-header-left">
          <MessageSquare size={16} />
          <span className="chat-title">Chat</span>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="chat-close-btn"
          title="Slide chat away"
        >
          <ChevronRight size={18} />
        </button>
      </div>

      {/* Messages Scroll Area */}
      <div className="messages-container">
        {messages.length === 0 ? (
          <div className="empty-chat">
            <p>No messages yet</p>
          </div>
        ) : (
          messages.map((msg) => {
            if (msg.isSystem) {
              return (
                <div key={msg.id} className="system-msg-row">
                  <span className="system-msg-text">{msg.message}</span>
                </div>
              );
            }

            const isMe = msg.userId === currentUser.id;

            return (
              <div
                key={msg.id}
                className={`message-row ${isMe ? 'is-me' : ''}`}
              >
                <img
                  src={msg.avatar || `https://api.dicebear.com/7.x/initials/svg?seed=${encodeURIComponent(msg.userName)}`}
                  alt={msg.userName}
                  className="msg-avatar"
                />

                <div className="msg-bubble-wrap">
                  <div className="msg-info">
                    <span className="msg-sender">{msg.userName}</span>
                    <span className="msg-time">{formatTime(msg.createdAt)}</span>
                  </div>

                  <div className="msg-bubble">
                    {msg.message}
                  </div>
                </div>
              </div>
            );
          })
        )}

        {/* Typing indicator */}
        {typingUsers.length > 0 && (
          <div className="typing-indicator">
            <span className="typing-dot" />
            <span>{typingUsers.join(', ')} typing...</span>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Chat Input */}
      <form onSubmit={handleSendMessage} className="chat-input-area">
        <div className="chat-input-wrapper">
          <input
            type="text"
            value={inputMessage}
            onChange={handleInputChange}
            placeholder="Type a message..."
            className="form-input chat-input-field"
          />
          <button
            type="submit"
            disabled={!inputMessage.trim()}
            className="chat-send-btn"
          >
            <Send size={14} />
          </button>
        </div>
      </form>
    </div>
  );
};
