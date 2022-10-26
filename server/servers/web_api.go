package servers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Doridian/wsvpn/shared/sockets"
)

type SocketStruct struct {
	ClientID   string `json:"client_id"`
	VPNIP      string `json:"vpn_ip"`
	LocalAddr  string `json:"local_addr"`
	RemoteAddr string `json:"remote_addr"`
}

const apiRouteSockets = "sockets"

func socketToJSON(clientID string, socket *sockets.Socket) SocketStruct {
	adapter := socket.GetAdapter()
	return SocketStruct{
		ClientID:   clientID,
		VPNIP:      socket.AssignedIP.String(),
		RemoteAddr: adapter.RemoteAddr().String(),
		LocalAddr:  adapter.LocalAddr().String(),
	}
}

func serveJSON(data interface{}, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(data)
}

func (s *Server) serveAPI(w http.ResponseWriter, r *http.Request, username string) {
	if !s.APIEnabled {
		http.Error(w, "API not enabled", http.StatusBadRequest)
		return
	}

	if len(s.APIUsers) > 0 && !s.APIUsers[username] {
		http.Error(w, "API access not allowed", http.StatusForbidden)
		return
	}

	pathSplit := strings.Split(r.URL.Path, "/")
	switch len(pathSplit) {
	case 3:
		switch pathSplit[2] {
		case apiRouteSockets:
			s.socketsLock.Lock()
			sockets := make([]SocketStruct, 0, len(s.sockets))
			for clientID, socket := range s.sockets {
				sockets = append(sockets, socketToJSON(clientID, socket))
			}
			s.socketsLock.Unlock()

			serveJSON(sockets, w)
			return
		}
	case 4:
		switch pathSplit[2] {
		case apiRouteSockets:
			clientID := pathSplit[3]
			s.socketsLock.Lock()
			socket := s.sockets[clientID]
			s.socketsLock.Unlock()

			if socket == nil {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}

			switch r.Method {
			case http.MethodGet:
				serveJSON(socketToJSON(clientID, socket), w)
				return
			case http.MethodDelete:
				socket.Close()
				http.Error(w, "OK", http.StatusOK)
				return
			}
		}
	}

	http.Error(w, "API method not implemented, yet", http.StatusNotFound)
}
