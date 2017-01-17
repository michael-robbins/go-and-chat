package gochat

type ChatServer struct {
	clients map[string]string
}

func NewChatServer() (*ChatServer, error) {
	return &ChatServer{}, nil
}

func (server ChatServer) Listen(connection_string string) error {
	// Bind to the IP/Port and listen for new incoming connections
	return nil
}

func (server *ChatServer) HandleIncomingConnection() error {
	return nil
}
