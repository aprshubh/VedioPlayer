import type { Room, RoomMember, User } from '../types';

const API_BASE = (import.meta.env.VITE_API_URL as string) || (window.location.origin.includes('5173') ? 'http://localhost:8080' : '');

export const getAuthToken = (): string | null => {
  return sessionStorage.getItem('dyuet_token');
};

export const setAuthToken = (token: string) => {
  sessionStorage.setItem('dyuet_token', token);
};

export const clearAuthToken = () => {
  sessionStorage.removeItem('dyuet_token');
  sessionStorage.removeItem('dyuet_user');
};

export const getStoredUser = (): User | null => {
  const data = sessionStorage.getItem('dyuet_user');
  if (!data) return null;
  try {
    return JSON.parse(data);
  } catch {
    return null;
  }
};

export const setStoredUser = (user: User) => {
  sessionStorage.setItem('dyuet_user', JSON.stringify(user));
};

const getHeaders = () => {
  const token = getAuthToken();
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };
};

export const api = {
  async requestOTP(email: string): Promise<{ success: boolean; message: string; code?: string }> {
    const res = await fetch(`${API_BASE}/api/auth/otp/request`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ email }),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  async verifyOTP(email: string, code: string, name?: string, avatar?: string): Promise<{ user: User; token: string }> {
    const res = await fetch(`${API_BASE}/api/auth/otp/verify`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ email, code, name, avatar }),
    });
    if (!res.ok) throw new Error(await res.text());
    const data = await res.json();
    setAuthToken(data.token);
    setStoredUser(data.user);
    return data;
  },

  async googleLogin(email: string, name: string, avatar: string): Promise<{ user: User; token: string }> {
    const res = await fetch(`${API_BASE}/api/auth/google`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ email, name, avatar }),
    });
    if (!res.ok) throw new Error(await res.text());
    const data = await res.json();
    setAuthToken(data.token);
    setStoredUser(data.user);
    return data;
  },

  async guestLogin(name: string): Promise<{ user: User; token: string }> {
    const res = await fetch(`${API_BASE}/api/auth/guest`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ name }),
    });
    if (!res.ok) throw new Error(await res.text());
    const data = await res.json();
    setAuthToken(data.token);
    setStoredUser(data.user);
    return data;
  },

  async createSession(name: string): Promise<{ user: User; token: string }> {
    return this.guestLogin(name);
  },

  async getMe(): Promise<User> {
    const res = await fetch(`${API_BASE}/api/auth/me`, {
      headers: getHeaders(),
    });
    if (!res.ok) throw new Error('Failed to fetch user');
    return res.json();
  },

  async createRoom(): Promise<Room> {
    const res = await fetch(`${API_BASE}/api/rooms`, {
      method: 'POST',
      headers: getHeaders(),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  async getRoomByCode(code: string): Promise<{ room: Room; members: RoomMember[] }> {
    const res = await fetch(`${API_BASE}/api/rooms/by-code/${code}`, {
      headers: getHeaders(),
    });
    if (!res.ok) throw new Error('Room not found');
    return res.json();
  },

  async joinRoom(code: string): Promise<{ room: Room; isHost: boolean }> {
    const res = await fetch(`${API_BASE}/api/rooms/join`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ code }),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },
};
