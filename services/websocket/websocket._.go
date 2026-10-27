package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	engine "github.com/zishang520/socket.io/servers/engine/v3"
	"github.com/zishang520/socket.io/servers/engine/v3/transports"
	"github.com/zishang520/socket.io/servers/socket/v3"
	"github.com/zishang520/socket.io/v3/pkg/types"
)

const (
	Path           = "/api/socket.io"
	pingInterval   = 25 * time.Second
	pingTimeout    = 20 * time.Second
	connectTimeout = 45 * time.Second
	maxPayload     = 1e6
	eventError     = "error"
	unauthorized   = "unauthorized"
	ClientKey      = "korda.websocket.client"
)

var errUnauthorized = errors.New(unauthorized)

type clientKey struct{}

type Identity struct {
	UserID    uuid.UUID
	SessionID uuid.NullUUID
	APIKeyID  uuid.NullUUID
	Client    string
}

type Authenticator func(r *http.Request) (*Identity, error)

type Observer func(room string, event string, args []json.RawMessage)

type Hub struct {
	mu           sync.RWMutex
	io           *socket.Server
	handler      http.Handler
	authenticate Authenticator
	connected    func(Identity)
	rejected     func(client string, err error)
	disconnected func(id Identity, reason string)
	observer     Observer
}

func New() (*Hub, error) {
	return new(Hub), nil
}

func (hub *Hub) SetAuthenticator(fn Authenticator) {
	hub.mu.Lock()
	hub.authenticate = fn
	hub.mu.Unlock()
}

func (hub *Hub) SetConnected(fn func(Identity)) {
	hub.mu.Lock()
	hub.connected = fn
	hub.mu.Unlock()
}

func (hub *Hub) SetRejected(fn func(client string, err error)) {
	hub.mu.Lock()
	hub.rejected = fn
	hub.mu.Unlock()
}

func (hub *Hub) SetDisconnected(fn func(id Identity, reason string)) {
	hub.mu.Lock()
	hub.disconnected = fn
	hub.mu.Unlock()
}

func (hub *Hub) Observe(fn Observer) {
	hub.mu.Lock()
	hub.observer = fn
	hub.mu.Unlock()
}

func (hub *Hub) Startup() error {
	opts := socket.DefaultServerOptions()
	opts.SetPath(Path)
	opts.SetServeClient(false)
	opts.SetPingInterval(pingInterval)
	opts.SetPingTimeout(pingTimeout)
	opts.SetMaxHttpBufferSize(maxPayload)
	opts.SetConnectTimeout(connectTimeout)
	opts.SetCors(&types.Cors{Origin: "*"})
	opts.SetTransports(types.NewSet[transports.TransportCtor](engine.WebSocket))

	io := socket.NewServer(nil, opts)
	if err := io.On("connection", func(clients ...any) {
		if s, ok := clients[0].(*socket.Socket); ok {
			hub.join(s)
		}
	}); err != nil {
		return err
	}

	hub.mu.Lock()
	hub.io = io
	hub.handler = io.ServeHandler(nil)
	hub.mu.Unlock()

	return nil
}

func (hub *Hub) Shutdown() error {
	hub.mu.Lock()
	io := hub.io
	hub.io = nil
	hub.mu.Unlock()

	if io != nil {
		io.Close(nil)
	}

	return nil
}

func (hub *Hub) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.mu.RLock()
		handler := hub.handler
		hub.mu.RUnlock()

		if handler == nil {
			http.NotFound(w, r)
			return
		}
		ctx := context.WithoutCancel(r.Context())
		if client, ok := r.Context().Value(ClientKey).(string); ok {
			ctx = context.WithValue(ctx, clientKey{}, client)
		}
		handler.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (hub *Hub) join(s *socket.Socket) {
	hub.mu.RLock()
	authenticate, connected, rejected, disconnected := hub.authenticate, hub.connected, hub.rejected, hub.disconnected
	hub.mu.RUnlock()

	r := s.Request().Request()
	client, _ := r.Context().Value(clientKey{}).(string)
	var id *Identity
	var err error
	if authenticate != nil {
		id, err = authenticate(r)
	}
	if authenticate == nil || err != nil || id == nil {
		if rejected != nil {
			if err == nil {
				err = errUnauthorized
			}
			rejected(client, err)
		}
		_ = s.Emit(eventError, unauthorized)
		s.Disconnect(false)
		return
	}
	id.Client = client

	s.Join(socket.Room(id.UserID.String()))
	if id.SessionID.Valid {
		s.Join(socket.Room(id.SessionID.UUID.String()))
	}
	if id.APIKeyID.Valid {
		s.Join(socket.Room(id.APIKeyID.UUID.String()))
	}
	if disconnected != nil {
		identity := *id
		_ = s.On("disconnect", func(args ...any) {
			reason := ""
			if len(args) > 0 {
				reason, _ = args[0].(string)
			}
			disconnected(identity, reason)
		})
	}
	if connected != nil {
		connected(*id)
	}
}

func encode(args []any) ([]any, []json.RawMessage) {
	out := make([]any, 0, len(args))
	raw := make([]json.RawMessage, 0, len(args))
	for _, arg := range args {
		encoded, err := json.Marshal(arg)
		if err != nil {
			continue
		}
		out = append(out, json.RawMessage(encoded))
		raw = append(raw, encoded)
	}

	return out, raw
}

func (hub *Hub) emit(room string, event string, args []any) {
	hub.mu.RLock()
	io, observer := hub.io, hub.observer
	hub.mu.RUnlock()

	values, raw := encode(args)
	if observer != nil {
		observer(room, event, raw)
	}
	if io == nil {
		return
	}
	if room == "" {
		io.Emit(event, values...)
		return
	}
	_ = io.To(socket.Room(room)).Emit(event, values...)
}

func (hub *Hub) ToUser(userID uuid.UUID, event string, args ...any) {
	hub.emit(userID.String(), event, args)
}

func (hub *Hub) ToSession(sessionID uuid.UUID, event string, args ...any) {
	hub.emit(sessionID.String(), event, args)
}

func (hub *Hub) Broadcast(event string, args ...any) {
	hub.emit("", event, args)
}

func (hub *Hub) disconnect(room string) {
	hub.mu.RLock()
	io := hub.io
	hub.mu.RUnlock()

	if io != nil {
		io.In(socket.Room(room)).DisconnectSockets(false)
	}
}

func (hub *Hub) DisconnectUser(userID uuid.UUID) {
	hub.disconnect(userID.String())
}

func (hub *Hub) DisconnectSession(sessionID uuid.UUID) {
	hub.disconnect(sessionID.String())
}

func (hub *Hub) DisconnectAPIKey(keyID uuid.UUID) {
	hub.disconnect(keyID.String())
}
