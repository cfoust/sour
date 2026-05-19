package sim

import (
	"context"
	"sync"

	"github.com/cfoust/sour/pkg/gameserver"
)

// Router demultiplexes the server's single outgoing channel to individual
// sim Clients. In production, the ingress layer does this; in simulation,
// we need to replicate it.
type Router struct {
	server  *gameserver.Server
	clients map[uint32]*Client // sessionID -> client
	mu      sync.RWMutex
	cancel  context.CancelFunc
	ctx     context.Context
	wg      sync.WaitGroup
}

// NewRouter creates a router that reads from the server's outgoing channel
// and dispatches packets to the appropriate Client.
func NewRouter(ctx context.Context, server *gameserver.Server) *Router {
	ctx, cancel := context.WithCancel(ctx)
	r := &Router{
		server:  server,
		clients: make(map[uint32]*Client),
		cancel:  cancel,
		ctx:     ctx,
	}
	r.wg.Add(1)
	go r.run()
	return r
}

func (r *Router) run() {
	defer r.wg.Done()
	outgoing := r.server.Outgoing()

	for {
		select {
		case <-r.ctx.Done():
			return
		case pkt, ok := <-outgoing:
			if !ok {
				return
			}
			r.mu.RLock()
			client, exists := r.clients[pkt.Session]
			r.mu.RUnlock()

			if exists {
				client.HandleMessages(pkt.Messages)
			}
		}
	}
}

// Add registers a Client with the router.
func (r *Router) Add(c *Client) {
	r.mu.Lock()
	r.clients[c.SessionID] = c
	r.mu.Unlock()
}

// Remove unregisters a Client from the router.
func (r *Router) Remove(c *Client) {
	r.mu.Lock()
	delete(r.clients, c.SessionID)
	r.mu.Unlock()
}

// Stop shuts down the router.
func (r *Router) Stop() {
	r.cancel()
	r.wg.Wait()
}
