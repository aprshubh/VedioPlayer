package main

import (
	"crypto/rand"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"watchparty-backend/internal/auth"
	"watchparty-backend/internal/model"
	"watchparty-backend/internal/store"
	"watchparty-backend/internal/video"
	ws "watchparty-backend/internal/websocket"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for dev / desktop app
	},
}

// Generate a random 6-character room code (e.g. 7XK92P)
func generateRoomCode() string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	bytes := make([]byte, 6)
	_, _ = rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = charset[int(b)%len(charset)]
	}
	return string(bytes)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dyuet-dev-secret-key-12345"
	}

	// 1. Initialize Storage Layer
	var dataStore store.Store
	pgConn := os.Getenv("DATABASE_URL")
	if pgConn != "" {
		log.Printf("Connecting to PostgreSQL at %s...", pgConn)
		pgStore, err := store.NewPostgresStore(pgConn)
		if err != nil {
			log.Fatalf("Failed to initialize PostgreSQL: %v", err)
		}
		dataStore = pgStore
		log.Println("Connected to PostgreSQL successfully.")
	} else {
		log.Println("DATABASE_URL not set: using high-performance In-Memory Store for instant zero-dependency execution.")
		dataStore = store.NewMemoryStore()
	}

	// 2. Initialize Services
	authService := auth.NewAuthService(jwtSecret, dataStore)
	videoSyncer := video.NewVideoSyncer(dataStore)
	hub := ws.NewHub(dataStore, videoSyncer)
	go hub.Run()

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "time": time.Now().String()})
	})

	// Auth: Request OTP
	mux.HandleFunc("/api/auth/otp/request", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
			http.Error(w, "Valid email required", http.StatusBadRequest)
			return
		}

		code, err := authService.RequestOTP(req.Email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("Generated OTP for %s: %s", req.Email, code)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "OTP sent successfully (Development code: 123456 or " + code + ")",
			"code":    code, // Included in response for seamless local testing
		})
	})

	// Auth: Verify OTP
	mux.HandleFunc("/api/auth/otp/verify", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Email  string `json:"email"`
			Code   string `json:"code"`
			Name   string `json:"name"`
			Avatar string `json:"avatar"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		user, token, err := authService.VerifyOTP(req.Email, req.Code, req.Name, req.Avatar)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"user":  user,
			"token": token,
		})
	})

	// Auth: Google Mock / Exchange
	mux.HandleFunc("/api/auth/google", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Email  string `json:"email"`
			Name   string `json:"name"`
			Avatar string `json:"avatar"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		user, token, err := authService.GoogleLogin(req.Email, req.Name, req.Avatar)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"user":  user,
			"token": token,
		})
	})

	// Auth: Guest Login
	mux.HandleFunc("/api/auth/guest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		user, token, err := authService.GuestLogin(req.Name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"user":  user,
			"token": token,
		})
	})

	// Auth: Current User Profile
	mux.HandleFunc("/api/auth/me", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := authService.ValidateToken(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := dataStore.GetUserByID(claims.UserID)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(user)
	})

	// Room: Create Room
	mux.HandleFunc("/api/rooms", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := authService.ValidateToken(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		roomCode := generateRoomCode()
		roomID := uuid.NewString()

		room := &model.Room{
			ID:                 roomID,
			Code:               roomCode,
			HostID:             claims.UserID,
			OnlyHostCanControl: false,
			CreatedAt:          time.Now(),
		}

		if err := dataStore.CreateRoom(room); err != nil {
			http.Error(w, "Failed to create room", http.StatusInternalServerError)
			return
		}

		// Add host as member
		user, _ := dataStore.GetUserByID(claims.UserID)
		if user == nil {
			user = &model.User{ID: claims.UserID, Name: claims.Name, Avatar: claims.Avatar}
		}
		_ = dataStore.AddMember(&model.RoomMember{
			RoomID:   room.ID,
			UserID:   claims.UserID,
			User:     *user,
			IsHost:   true,
			JoinedAt: time.Now(),
			IsOnline: true,
		})

		// Initialize video state
		_ = dataStore.SetVideoState(&model.VideoState{
			RoomID:    room.ID,
			Playing:   false,
			Position:  0,
			Rate:      1.0,
			UpdatedAt: time.Now().UnixMilli(),
		})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(room)
	})

	// Room: Get Room by Code
	mux.HandleFunc("/api/rooms/by-code/", func(w http.ResponseWriter, r *http.Request) {
		code := strings.TrimPrefix(r.URL.Path, "/api/rooms/by-code/")
		code = strings.ToUpper(strings.TrimSpace(code))
		room, err := dataStore.GetRoomByCode(code)
		if err != nil {
			http.Error(w, "Room not found", http.StatusNotFound)
			return
		}

		members, _ := dataStore.GetRoomMembers(room.ID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"room":    room,
			"members": members,
		})
	})

	// Room: Join Room by Code
	mux.HandleFunc("/api/rooms/join", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := authService.ValidateToken(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			Code string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
		room, err := dataStore.GetRoomByCode(req.Code)
		if err != nil {
			http.Error(w, "Room not found", http.StatusNotFound)
			return
		}

		user, _ := dataStore.GetUserByID(claims.UserID)
		if user == nil {
			user = &model.User{ID: claims.UserID, Name: claims.Name, Avatar: claims.Avatar}
		}

		isHost := (room.HostID == claims.UserID)
		_ = dataStore.AddMember(&model.RoomMember{
			RoomID:   room.ID,
			UserID:   claims.UserID,
			User:     *user,
			IsHost:   isHost,
			JoinedAt: time.Now(),
			IsOnline: true,
		})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"room":   room,
			"isHost": isHost,
		})
	})

	// WebSocket Endpoint: /ws?token=...&roomId=...
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		roomID := r.URL.Query().Get("roomId")

		if token == "" || roomID == "" {
			http.Error(w, "Token and roomId query parameters are required", http.StatusBadRequest)
			return
		}

		claims, err := authService.ValidateToken(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		room, err := dataStore.GetRoomByID(roomID)
		if err != nil {
			http.Error(w, "Room not found", http.StatusNotFound)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade failed: %v", err)
			return
		}

		client := &ws.Client{
			Hub:      hub,
			Conn:     conn,
			Send:     make(chan []byte, 256),
			UserID:   claims.UserID,
			UserName: claims.Name,
			Avatar:   claims.Avatar,
			RoomID:   room.ID,
			IsHost:   (room.HostID == claims.UserID),
		}

		hub.Register <- client

		go client.WritePump()
		go client.ReadPump()
	})

	serverAddr := ":" + port
	log.Printf("🚀 Dyuet Go backend starting on http://localhost%s\n", serverAddr)
	log.Printf("⚡ WebSocket endpoint ready at ws://localhost%s/ws\n", serverAddr)

	handler := corsMiddleware(mux)
	if err := http.ListenAndServe(serverAddr, handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
