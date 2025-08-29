package peers

import "sync"

type Peer struct {
	ID   string `json:"id"`
	Addr string `json:"addr"`
}

type Store struct {
	mu    sync.RWMutex
	peers map[string]Peer
}

func NewStore() *Store {
	return &Store{
		peers: make(map[string]Peer),
	}
}

func (s *Store) Add(p Peer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.peers[p.ID] = p
}

func (s *Store) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.peers, id)
}

func (s *Store) List() []Peer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Peer, 0, len(s.peers))
	for _, p := range s.peers {
		out = append(out, p)
	}
	return out
}

func (s *Store) Exists(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.peers[id]
	return ok
}
