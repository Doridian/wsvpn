package servers

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Doridian/water"
	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/Doridian/wsvpn/shared/features"
	"github.com/Doridian/wsvpn/shared/iface"
	"github.com/Doridian/wsvpn/shared/sockets"
	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

func (s *Server) serveSocket(w http.ResponseWriter, r *http.Request) {
	clientUUID, err := uuid.NewRandom()
	if err != nil {
		s.log.Printf("Error creating client ID: %v", err)
		http.Error(w, "Error creating client ID", http.StatusInternalServerError)
		return
	}
	clientID := clientUUID.String()
	clientLogger := shared.MakeLogger("CLIENT", clientID)

	for key, values := range s.headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	tlsConnectionState := r.TLS

	http3Hijacker, ok := w.(http3.Hijacker)
	if ok {
		qconn, ok := http3Hijacker.StreamCreator().(quic.Connection)
		if ok {
			qtlsState := qconn.ConnectionState().TLS
			tlsConnectionState = &qtlsState
		}
	}

	if r.URL.Path == rootRoutePreauthorize {
		s.handlePreauthorize(clientLogger, w, r, tlsConnectionState)
		return
	}

	if tlsConnectionState != nil {
		clientLogger.Printf("TLS %s connection established with cipher=%s", shared.TLSVersionString(tlsConnectionState.Version), tls.CipherSuiteName(tlsConnectionState.CipherSuite))
	} else {
		clientLogger.Printf("Unencrypted connection established")
	}

	var authOk bool
	var authUsername string
	if len(s.PreauthorizeSecret) > 0 && strings.HasPrefix(r.URL.Path, prefixRoutePreauthorize) {
		preauthToken := r.URL.Path[len(prefixRoutePreauthorize):]
		authOk, authUsername = s.handlePreauthorizeToken(clientLogger, w, r, preauthToken)
	} else {
		authOk, authUsername = s.handleSocketAuth(clientLogger, w, r, tlsConnectionState)
	}

	if !authOk {
		return
	}

	if authUsername != "" {
		clientLogger.Printf("Authenticated as: %s", authUsername)
	}

	var adapter adapters.SocketAdapter
	wasUpgraded := false
	for _, upgrader := range s.upgraders {
		if !upgrader.Matches(r) {
			continue
		}
		wasUpgraded = true
		adapter, err = upgrader.Upgrade(w, r)
		if err != nil {
			clientLogger.Printf("Error upgrading connection: %v", err)
			return
		}
		break
	}

	if !wasUpgraded {
		s.serveHTTP(w, r, authUsername)
		return
	}

	defer adapter.Close()
	s.addCloser(adapter)

	clientLogger.Printf("Upgraded connection to %s", adapter.Name())

	var slot uint64 = 1
	maxSlot := s.VPNNet.GetClientSlots() + 1
	s.slotMutex.Lock()
	for s.usedSlots[slot] {
		slot = slot + 1
		if slot > maxSlot {
			s.slotMutex.Unlock()
			clientLogger.Println("Cannot connect new client: IP slots exhausted")
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

	ipClient, err := s.VPNNet.GetIPAt(int(slot) + 1)
	if err != nil {
		clientLogger.Printf("Error transforming client IP: %v", err)
		return
	}

	clientLogger.Printf("Command serialization: %s", commands.SerializationTypeToString(adapter.GetCommandSerializationType()))

	var ifaceManaged bool
	var localIface *iface.WaterInterfaceWrapper

	if s.InterfaceConfig.OneInterfacePerConnection {
		ifaceManaged = true

		s.ifaceCreationMutex.Lock()
		ifaceConfig := water.Config{
			DeviceType: s.Mode.ToWaterDeviceType(),
		}
		err = iface.GetPlatformSpecifics(&ifaceConfig, s.InterfaceConfig)
		if err != nil {
			s.ifaceCreationMutex.Unlock()
			clientLogger.Printf("Error extending iface config: %v", err)
			return
		}

		var localIfaceW *water.Interface
		localIfaceW, err = water.New(ifaceConfig)
		s.ifaceCreationMutex.Unlock()
		if err != nil {
			clientLogger.Printf("Error creating new iface: %v", err)
			return
		}

		localIface = iface.NewInterfaceWrapper(localIfaceW)

		defer localIface.Close()

		clientLogger.Printf("Assigned interface %s", localIfaceW.Name())

		if s.DoLocalIPConfig {
			err = localIface.Configure(s.VPNNet.GetServerIP(), nil, ipClient)
		} else {
			err = localIface.Configure(nil, nil, nil)
		}
		if err != nil {
			clientLogger.Printf("Error configuring interface: %v", err)
			return
		}
		err = localIface.SetMTU(s.mtu)
		if err != nil {
			clientLogger.Printf("Error setting interface MTU: %v", err)
			return
		}
	} else {
		ifaceManaged = false
		localIface = s.mainIface
	}

	remoteNetStr := fmt.Sprintf("%s/%d", ipClient.String(), s.VPNNet.GetSize())
	ifaceName := localIface.Interface.Name()

	doRunEventScript := func(event string) {
		eventErr := s.RunEventScript(event, remoteNetStr, ifaceName, authUsername)
		if eventErr != nil {
			s.log.Printf("Error in %s script: %v", event, eventErr)
		}
	}

	socket := sockets.MakeSocket(clientLogger, adapter, localIface, ifaceManaged, doRunEventScript)
	socket.Metadata["username"] = authUsername
	defer socket.Close()

	maxConns := s.MaxConnectionsPerUser

	s.socketsLock.Lock()
	if authUsername != "" && maxConns > 0 {
		userSocks := s.authenticatedSockets[authUsername]

		if userSocks != nil && len(userSocks) >= maxConns {
			switch s.MaxConnectionsPerUserMode {
			case MaxConnectionsPerUserKillOldest:
				toKill := userSocks[0]
				userSocks = userSocks[1:]
				toKill.CloseError(errors.New("maximum connections for user exceeded"))
			case MaxConnectionsPerUserPreventNew:
				s.socketsLock.Unlock()
				socket.CloseError(errors.New("maximum connections for user exceeded"))
				return
			}
		}

		if userSocks == nil {
			userSocks = []*sockets.Socket{socket}
		} else {
			userSocks = append(userSocks, socket)
		}
		s.authenticatedSockets[authUsername] = userSocks
	}
	s.sockets[clientID] = socket
	s.socketsLock.Unlock()

	defer func() {
		s.socketsLock.Lock()
		delete(s.sockets, clientID)

		if authUsername != "" {
			userSocks := s.authenticatedSockets[authUsername]

			if userSocks != nil {
				newSocks := make([]*sockets.Socket, 0)

				for _, sock := range s.authenticatedSockets[authUsername] {
					if sock == socket {
						continue
					}
					newSocks = append(newSocks, sock)
				}

				if len(newSocks) == 0 {
					delete(s.authenticatedSockets, authUsername)
				} else {
					s.authenticatedSockets[authUsername] = newSocks
				}
			}
		}
		s.socketsLock.Unlock()
	}()

	for feat, en := range s.localFeatures {
		socket.SetLocalFeature(feat, en)
	}

	socket.AssignedIP = ipClient

	if s.SocketConfigurator != nil {
		err = s.SocketConfigurator.ConfigureSocket(socket)
		if err != nil {
			socket.CloseError(fmt.Errorf("error configuring socket: %v", err))
			return
		}
	}

	if s.PacketHandler != nil {
		socket.SetPacketHandler(s.PacketHandler)
	}
	socket.SetMTU(s.mtu)

	clientLogger.Println("Connection fully established")
	defer clientLogger.Println("Disconnected")

	socket.Serve()
	socket.WaitReady()

	err = socket.MakeAndSendCommand(&commands.InitParameters{
		ClientID:            clientID,
		ServerID:            s.serverID,
		Mode:                s.Mode.ToString(),
		DoIPConfig:          s.DoRemoteIPConfig,
		IPAddress:           remoteNetStr,
		MTU:                 s.mtu,
		EnableFragmentation: socket.IsLocalFeature(features.Fragmentation),
	})
	if err != nil {
		socket.CloseError(fmt.Errorf("error sending init command: %v", err))
		return
	}

	socket.Wait()
}

func (s *Server) UpdateSocketConfig() error {
	if s.SocketConfigurator == nil {
		return nil
	}

	s.socketsLock.Lock()
	defer s.socketsLock.Unlock()
	for _, socket := range s.sockets {
		err := s.SocketConfigurator.ConfigureSocket(socket)
		if err != nil {
			return err
		}
	}

	return nil
}
