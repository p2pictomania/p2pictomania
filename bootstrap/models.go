package bootstrap

//PeerList is a structure that holds the list of peers returned by the Bootstrap Node
type PeerList struct {
	PeerIPs []string `json:"peerIPs"`
}
