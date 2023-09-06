package lobby

type LobbyRoom struct {
	clients map[*Client]bool
}
