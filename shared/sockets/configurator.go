package sockets

type SocketConfigurator interface {
	ConfigureSocket(sock *Socket) error
}
