package serv

import (
	"net"
	"net/http"
	"sync"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/sirupsen/logrus"
)

// Server is a websocket server.
type Server struct {
	once    sync.Once
	id      string
	address string
	sync.Mutex
	// 会话列表
	users map[string]net.Conn
}

func NewServer(id, address string) *Server {
	return newServer(id, address)
}

func newServer(id, address string) *Server {
	return &Server{
		id:      id,
		address: address,
		users:   make(map[string]net.Conn, 100),
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	log := logrus.WithFields(logrus.Fields{
		"module": "Server",
		"listen": s.address,
		"id":     s.id,
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 1.升级
		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			conn.Close()
			return
		}

		// 2.
	})

}
