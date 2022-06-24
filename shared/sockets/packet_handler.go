package sockets

type PacketHandler interface {
	HandlePacket(socket *Socket, packet []byte) (bool, error)
	RegisterSocket(socket *Socket)
	UnregisterSocket(socket *Socket)
}
