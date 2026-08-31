export interface User {
  id: string;
  name: string;
  email: string;
  avatar: string;
  createdAt?: string;
}

export interface Room {
  id: string;
  code: string;
  hostId: string;
  onlyHostCanControl: boolean;
  createdAt: string;
}

export interface RoomMember {
  roomId: string;
  userId: string;
  user: User;
  isHost: boolean;
  isOnline: boolean;
  joinedAt: string;
}

export interface Message {
  id: string;
  roomId: string;
  userId: string;
  userName: string;
  avatar?: string;
  message: string;
  isSystem: boolean;
  createdAt: string;
}

export interface VideoState {
  roomId: string;
  playing: boolean;
  position: number; // seconds
  rate: number;
  updatedAt: number; // ms timestamp
  changedBy?: string;
}

export interface SyncCorrection {
  playing: boolean;
  serverPosition: number;
  rate: number;
  drift: number;
  serverTimestamp: number;
  action: 'NONE' | 'RATE_ADJUST' | 'HARD_SEEK';
  targetRate: number;
}

export type EventType =
  | 'VIDEO_PLAY'
  | 'VIDEO_PAUSE'
  | 'VIDEO_SEEK'
  | 'VIDEO_RATE'
  | 'SYNC_REQUEST'
  | 'SYNC_CORRECTION'
  | 'CHAT_MESSAGE'
  | 'USER_JOIN'
  | 'USER_LEAVE'
  | 'PRESENCE_UPDATE'
  | 'TYPING'
  | 'ROOM_STATE'
  | 'UPDATE_SETTINGS'
  | 'AUDIO_CHANGE_REQUEST'
  | 'ERROR';

export interface WSMessage<T = unknown> {
  type: EventType;
  roomId?: string;
  userId?: string;
  payload: T;
  timestamp: number;
}

export interface AudioChangePayload {
  trackIndex: number;
  trackLabel: string;
  fromUser: string;
}
