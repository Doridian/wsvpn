package servers

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/Doridian/wsvpn/shared/sockets"
	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/google/uuid"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
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

	tlsConnectionState := r.TLS

	http3Hijacker, ok := w.(http3.Hijacker)
	if ok {
		qconn, ok := http3Hijacker.StreamCreator().(quic.Connection)
		if ok {
			qlsState := qconn.ConnectionState().TLS.ConnectionState
			tlsConnectionState = &qlsState
		}
	}

	if tlsConnectionState != nil {
		clientLogger.Printf("TLS %s connection established with cipher=%s", shared.TlsVersionString(tlsConnectionState.Version), tls.CipherSuiteName(tlsConnectionState.CipherSuite))
	} else {
		clientLogger.Printf("Unencrypted connection established")
	}

	authOk, authUsername := s.handleSocketAuth(clientLogger, w, r, tlsConnectionState)
	if !authOk {
		return
	}

	if authUsername != "" {
		clientLogger.Printf("Authenticated as: %s", authUsername)
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
	err = errors.New("no matching upgrader")
	for _, upgrader := range s.upgraders {
		if !upgrader.Matches(r) {
			continue
		}
		adapter, err = upgrader.Upgrade(w, r)
		break
	}

	if err != nil {
		clientLogger.Printf("Error upgrading connection: %v", err)
		return
	}

	defer adapter.Close()
	s.addCloser(adapter)

	clientLogger.Printf("Upgraded connection to %s", adapter.Name())

	ipClient, err := s.VPNNet.GetIPAt(int(slot) + 1)
	if err != nil {
		clientLogger.Printf("Error transforming client IP: %v", err)
		return
	}

	clientLogger.Printf("Command serialization: %s", commands.SerializationTypeToString(adapter.GetCommandSerializationType()))

	var ifaceManaged bool
	var iface *water.Interface

	if s.InterfaceConfig.OneInterfacePerConnection {
		ifaceManaged = true

		s.ifaceCreationMutex.Lock()
		ifaceConfig := water.Config{
			DeviceType: s.Mode.ToWaterDeviceType(),
		}
		err = s.getPlatformSpecifics(&ifaceConfig)
		if err != nil {
			s.ifaceCreationMutex.Unlock()
			clientLogger.Printf("Error extending iface config: %v", err)
			return
		}

		iface, err = water.New(ifaceConfig)
		s.ifaceCreationMutex.Unlock()
		if err != nil {
			clientLogger.Printf("Error creating new iface: %v", err)
			return
		}

		defer iface.Close()

		clientLogger.Printf("Assigned interface %s", iface.Name())

		err = s.configIface(iface, ipClient)
		if err != nil {
			clientLogger.Printf("Error configuring interface: %v", err)
			return
		}
	} else {
		ifaceManaged = false
		iface = s.mainIface
	}

	socket := sockets.MakeSocket(clientLogger, adapter, iface, ifaceManaged)
	defer socket.Close()

	socket.AssignedIP = shared.NetIPToIPv4(ipClient)

	if s.SocketConfigurator != nil {
		err := s.SocketConfigurator.ConfigureSocket(socket)
		if err != nil {
			socket.CloseError(fmt.Errorf("error configuring socket: %v", err))
			return
		}
	}
	socket.SetAllowEnableFragmentation(s.EnableFragmentation)
	if s.PacketHandler != nil {
		socket.SetPacketHandler(s.PacketHandler)
	}
	socket.SetMTU(s.mtu)

	clientLogger.Println("Connection fully established")
	defer clientLogger.Println("Disconnected")

	socket.Serve()
	socket.WaitReady()

	remoteNetStr := fmt.Sprintf("%s/%d", ipClient.String(), s.VPNNet.GetSize())
	ifaceName := iface.Name()

	err = socket.MakeAndSendCommand(&commands.InitParameters{
		ClientID:            clientId,
		ServerID:            s.serverId,
		Mode:                s.Mode.ToString(),
		DoIpConfig:          s.DoRemoteIpConfig,
		IpAddress:           remoteNetStr,
		MTU:                 s.mtu,
		EnableFragmentation: s.EnableFragmentation,
	})
	if err != nil {
		socket.CloseError(fmt.Errorf("error sending init command: %v", err))
		return
	}

	doRunEventScript := func(event string) {
		err := s.RunEventScript(event, remoteNetStr, ifaceName, authUsername)
		if err != nil {
			s.log.Printf("Error in %s script: %v", event, err)
		}
	}

	go doRunEventScript(shared.EventUp)
	socket.Wait()
	go doRunEventScript(shared.EventDown)
}
