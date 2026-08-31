# 🎬 Dyuet — Synchronized Local Video Player & Watch Party

**Dyuet** is a high-precision, ultra-low-bandwidth watch party platform. It synchronizes local video playback (e.g. `C:\Movies\movie.mp4`) across multiple devices in real-time with integrated live chat, room permissions, online presence detection, and millisecond drift correction — all with **zero video upload or server video bandwidth**.

---

## 📁 Clean Two-Folder Structure

```
dyuet/
├── frontend/             # Website (React + TypeScript + Vite + Tailwind CSS)
│   ├── src/
│   │   ├── auth/         # AuthModal (Google, OTP, Quick presets)
│   │   ├── chat/         # ChatPanel (Live chat, typing indicator, emojis)
│   │   ├── room/         # RoomControls (Room code, presence, host toggle)
│   │   ├── services/     # REST API client & WebSocket manager
│   │   ├── types/        # TypeScript data contracts
│   │   ├── video/        # VideoPlayer (HTML5 local player & drift HUD)
│   │   ├── App.tsx       # Main cinema application layout
│   │   └── index.css     # Dark cinema theme styling
│   ├── package.json
│   └── vite.config.ts
│
├── backend/              # Go Backend (WebSocket + REST API)
│   ├── internal/
│   │   ├── auth/         # JWT generation & email OTP validation
│   │   ├── model/        # WS event packets & domain models
│   │   ├── store/        # In-Memory Store & PostgreSQL/Redis drivers
│   │   ├── video/        # Drift calculation engine & rate nudging
│   │   └── websocket/    # Hub, client pumps, and room pub/sub
│   ├── Dockerfile
│   ├── go.mod
│   └── main.go           # Server entry point
│
├── migrations/           # PostgreSQL schema (001_init.sql)
├── docker-compose.yml    # Docker setup for Postgres + Redis + Go
├── start-all.ps1         # 1-Click launcher (starts both and opens browser)
├── start-backend.ps1     # Starts backend on :8080
└── start-frontend.ps1    # Starts frontend on :5173
```

---

## 🚀 How to Run

### 1-Click Launch (Windows PowerShell)
In the root folder:
```powershell
.\start-all.ps1
```
This automatically starts the Go backend on `http://localhost:8080`, launches the website on `http://localhost:5173`, and opens the browser.

### Manual Launch

#### 1. Backend:
```powershell
cd backend
go run main.go
```
*Listens on `http://localhost:8080` (WebSocket at `ws://localhost:8080/ws`)*

#### 2. Website (Frontend):
```powershell
cd frontend
npm run dev
```
*Available at `http://localhost:5173`*

---

## 🎬 How Dyuet Works

1. **Zero Video Bandwidth**:
   - Movie files stay 100% on your machine.
   - User A selects `C:\Movies\movie.mp4` via file picker or drag-and-drop.
   - User B selects their local copy of `movie.mp4`.
   - Video is loaded as an HTML5 `blob:` URL. The server **never** touches or streams video files!

2. **High-Precision Drift Sync**:
   - When any authorized member clicks Play, Pause, or seeks, a lightweight JSON event is sent over WebSockets.
   - Every 3 seconds, clients verify drift with the server's authoritative clock.
   - **Small drift (100ms - 700ms)**: The client dynamically micro-adjusts playback rate (`0.97x` – `1.03x`) to catch up smoothly without audio pops or seeking glitches.
   - **Large drift (> 700ms)**: The client snaps directly to the server position.

3. **👑 Host vs Everyone Control**:
   - Toggle control between **Everyone** and **👑 Host Only** directly from the top bar.

4. **💬 Real-Time Chat & Presence**:
   - Integrated room chat with typing indicator, sender avatars, system playback events, and quick emoji bar.
   - 🟢 Online / Offline presence status for all room members.
