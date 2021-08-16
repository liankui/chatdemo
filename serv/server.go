package serv

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/gobwas/ws"
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
		conn, _, _, err := ws.UpgradeHTTP(r, w)	// 通过websocket库来为一个websocket长连接
		if err != nil {
			conn.Close()
			return
		}

		// 2.读取userId
		user := r.URL.Query().Get("user")
		if user == "" {
			conn.Close()
			return
		}

		// 3.添加到会话管理中
		old, ok := s.addUser(user, conn)
		if ok {
			old.Close() // 断开旧的连接
		}

		log.Infof("user %s in", user)

		go func() {
			// 4.读取消息
			s.readloop(user, conn)

			conn.Close()
			// 5.断开连接，删除用户
			s.delUser(user)

			log.Infof("connection of %s closed", user)
		}()
	})

	log.Infoln("started")
	return http.ListenAndServe(s.address, mux)
}

// 同账号互踢
func (s *Server) addUser(user string, conn net.Conn) (net.Conn, bool) {
	s.Lock()
	defer s.Unlock()
	old, ok := s.users[user] // 返回旧的连接
	s.users[user] = conn     // 缓存
	return old, ok
}

func (s *Server) readloop(user string, conn net.Conn) error {
	for {
		frame, err := ws.ReadFrame(conn)	// 从TCP缓冲中读取一帧的消息
		if err != nil {
			return err
		}

		if frame.Header.OpCode == ws.OpClose {
			return errors.New("remote side close the conn")
		}

		if frame.Header.Masked {
			ws.Cipher(frame.Payload, frame.Header.Mask, 0)
		}
	}
}

func (s *Server) delUser(user string) {
	s.Lock()
	defer s.Unlock()
	delete(s.users, user)
}

func (s *Server) Shutdown() {
	s.once.Do(func() {
		s.Lock()
		defer s.Unlock()
		for _, conn := range s.users {
			conn.Close()
		}
	})
}

// 全网广播
func (s *Server) handle(user string, message string) {
	logrus.Infof("recv message %s from %s", message, user)

	s.Lock()
	defer s.Unlock()
	broadcast := fmt.Sprintf("%s -- from %s", message, user)
	for u, conn := range s.users {
		if u == user {
			continue
		}

		logrus.Infof("send to %s : %s", u, broadcast)
		err := s.writeText(conn, broadcast)
		if err != nil {
			logrus.Infof("write to %s failed, error: %v", user, err)
		}
	}
}

func (s *Server) writeText(conn net.Conn, message string) error {
	// 创建文本帧数据
	f := ws.NewTextFrame([]byte(message))
	return ws.WriteFrame(conn, f)
}
