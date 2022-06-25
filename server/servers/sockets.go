package servers

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/Doridian/wsvpn/shared/sockets"
	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/google/uuid"
	"github.com/songgao/water"
)

func (s *Server) serveSocket(w http.ResponseWriter, r *http.Request) {
	var err error

	clientIdUUID, err := uuid.NewRandom()
	if err != nil {
		s.log.Printf("Error creating client ID: %v", err)
		http.Error(w, "Error creating client ID", http.StatusInternalServerError)
		return
	}

	clientId := clientIdUUID.String()

	clientLogger := shared.MakeLogger("CLIENT", clientId)

	if r.TLS != nil {
		clientLogger.Printf("TLS %s connection established with cipher=%s", shared.TlsVersionString(r.TLS.Version), tls.CipherSuiteName(r.TLS.CipherSuite))
	} else {
		clientLogger.Printf("Unencrypted connection established")
	}

	authOk := s.handleSocketAuth(clientLogger, w, r)
	if !authOk {
		return
	}

	var slot uint64 = 1
	maxSlot := s.VPNNet.GetClientSlots() + 1
	s.slotMutex.Lock()
	for s.usedSlots[slot] {
		slot = slot + 1
		if slot > maxSlot {
			s.slotMutex.Unlock()
			clientLogger.Println("Cannot connect new client: IP slots exhausted")
			http.Error(w, "IP slots exhausted", http.StatusInternalServerError)
			return
		}
	}
	s.usedSlots[slot] = true
	s.slotMutex.Unlock()

	defer func() {
		s.slotMutex.Lock()
		delete(s.usedSlots, slot)
		s.slotMutex.Unlock()
	}()

	var adapter adapters.SocketAdapter
	if r.Proto == "webtransport" && s.HTTP3Enabled {
		adapter, err = s.serveWebTransport(w, r)
	} else {
		adapter, err = s.serveWebSocket(w, r)
	}

	if err != nil {
		clientLogger.Printf("Error upgrading connection: %v", err)
		return
	}

	defer adapter.Close()

	clientLogger.Printf("Upgraded connection to %s", adapter.Name())

	ipClient, err := s.VPNNet.GetIPAt(int(slot) + 1)
	if err != nil {
		clientLogger.Printf("Error transforming client IP: %v", err)
		return
	}

	var iface *water.Interface
	if s.Mode == shared.VPN_MODE_TAP {
		iface = s.tapIface
	} else {
		s.ifaceCreationMutex.Lock()
		tunConfig := water.Config{
			DeviceType: water.TUN,
		}
		err = s.extendTUNConfig(&tunConfig)
		if err != nil {
			s.ifaceCreationMutex.Unlock()
			clientLogger.Printf("Error extending TUN config: %v", err)
			return
		}

		iface, err = water.New(tunConfig)
		s.ifaceCreationMutex.Unlock()
		if err != nil {
			clientLogger.Printf("Error creating new TUN: %v", err)
			return
		}

		defer iface.Close()

		clientLogger.Printf("Assigned interface %s", iface.Name())

		err = s.configIface(iface, ipClient)
		if err != nil {
			clientLogger.Printf("Error configuring interface: %v", err)
			return
		}
	}

	socket := sockets.MakeSocket(clientLogger, adapter, iface, s.Mode == shared.VPN_MODE_TUN)
	defer socket.Close()

	if s.SocketConfigurator != nil {
		err := s.SocketConfigurator.ConfigureSocket(socket)
		if err != nil {
			clientLogger.Printf("Error configuring socket: %v", err)
			http.Error(w, "Error configuring socket", http.StatusInternalServerError)
			return
		}
	}
	if s.SocketGroup != nil {
		socket.SetPacketHandler(s.SocketGroup)
	}
	socket.SetMTU(s.mtu)

	clientLogger.Println("Connection fully established")
	defer clientLogger.Println("Disconnected")

	socket.Serve()
	socket.MakeAndSendCommand(&commands.InitParameters{
		ClientID:   clientId,
		ServerID:   s.serverId,
		Mode:       s.Mode.ToString(),
		DoIpConfig: s.DoRemoteIpConfig,
		IpAddress:  fmt.Sprintf("%s/%d", ipClient.String(), s.VPNNet.GetSize()),
		MTU:        s.mtu,
	})
	socket.Wait()
}
