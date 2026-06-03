package chat

import (
	"encoding/json"
	"sync"
	"time"

	"auto-store-api/internal/models"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	redisChannelPrefix = "chat:conv:"
	writeWait          = 10 * time.Second
	pongWait           = 60 * time.Second
	pingPeriod         = 30 * time.Second
)

type WSOutbound struct {
	Type    string              `json:"type"`
	Message *models.ChatMessage `json:"message,omitempty"`
	Code    string              `json:"code,omitempty"`
	Error   string              `json:"error,omitempty"`
}

// MessageHandler persists an inbound WebSocket chat message.
type MessageHandler func(conversationID uuid.UUID, body string, isAdmin bool, userID, guestID *uuid.UUID) (*models.ChatMessage, error)

type client struct {
	hub            *Hub
	conn           *websocket.Conn
	send           chan []byte
	conversationID uuid.UUID
	isAdmin        bool
	userID         *uuid.UUID
	guestID        *uuid.UUID
	canAccess      AccessChecker
	onMessage      MessageHandler
}

// AccessChecker validates subscribe access for a conversation.
type AccessChecker func(conversationID uuid.UUID, isAdmin bool, userID, guestID *uuid.UUID) bool

type Hub struct {
	mu          sync.RWMutex
	clients     map[*client]struct{}
	subscribers map[uuid.UUID]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients:     make(map[*client]struct{}),
		subscribers: make(map[uuid.UUID]map[*client]struct{}),
	}
}

func (h *Hub) PublishMessage(conversationID uuid.UUID, message *models.ChatMessage) {
	payload, err := json.Marshal(WSOutbound{
		Type:    "message.new",
		Message: message,
	})
	if err != nil {
		return
	}
	h.Broadcast(conversationID, payload)
}

func (h *Hub) Broadcast(conversationID uuid.UUID, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	subs := h.subscribers[conversationID]
	for c := range subs {
		select {
		case c.send <- payload:
		default:
			go h.removeClient(c)
		}
	}
}

func (h *Hub) Subscribe(c *client, conversationID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	c.conversationID = conversationID
	if h.subscribers[conversationID] == nil {
		h.subscribers[conversationID] = make(map[*client]struct{})
	}
	h.subscribers[conversationID][c] = struct{}{}
}

func (h *Hub) Register(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
}

func (h *Hub) removeClient(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; !ok {
		return
	}
	delete(h.clients, c)
	if c.conversationID != uuid.Nil {
		if subs, ok := h.subscribers[c.conversationID]; ok {
			delete(subs, c)
			if len(subs) == 0 {
				delete(h.subscribers, c.conversationID)
			}
		}
	}
	close(c.send)
	_ = c.conn.Close()
}

func (h *Hub) HandleConnection(conn *websocket.Conn, isAdmin bool, userID, guestID *uuid.UUID, canAccess AccessChecker, onMessage MessageHandler) {
	c := &client{
		hub:       h,
		conn:      conn,
		send:      make(chan []byte, 16),
		isAdmin:   isAdmin,
		userID:    userID,
		guestID:   guestID,
		canAccess: canAccess,
		onMessage: onMessage,
	}
	h.Register(c)
	go c.writePump()
	c.readPump()
}

func (c *client) readPump() {
	defer func() {
		c.hub.removeClient(c)
	}()
	c.conn.SetReadLimit(4096)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var frame struct {
			Type           string    `json:"type"`
			ConversationID uuid.UUID `json:"conversation_id"`
			Body           string    `json:"body"`
		}
		if json.Unmarshal(data, &frame) != nil {
			continue
		}
		switch frame.Type {
		case "subscribe":
			if frame.ConversationID == uuid.Nil {
				continue
			}
			if c.canAccess != nil && !c.canAccess(frame.ConversationID, c.isAdmin, c.userID, c.guestID) {
				c.sendError("forbidden", "forbidden")
				continue
			}
			c.hub.Subscribe(c, frame.ConversationID)
		case "message":
			if frame.ConversationID == uuid.Nil || frame.Body == "" {
				c.sendError("bad_request", "conversation_id and body are required")
				continue
			}
			if c.canAccess != nil && !c.canAccess(frame.ConversationID, c.isAdmin, c.userID, c.guestID) {
				c.sendError("forbidden", "forbidden")
				continue
			}
			if c.onMessage == nil {
				c.sendError("unavailable", "message send not configured")
				continue
			}
			msg, err := c.onMessage(frame.ConversationID, frame.Body, c.isAdmin, c.userID, c.guestID)
			if err != nil {
				c.sendError("error", err.Error())
				continue
			}
			// SendMessage publishes message.new to all subscribers (including this client).
			_ = msg
		}
	}
}

func (c *client) sendError(code, message string) {
	errPayload, _ := json.Marshal(WSOutbound{Type: "error", Code: code, Error: message})
	select {
	case c.send <- errPayload:
	default:
	}
}

func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.hub.removeClient(c)
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
