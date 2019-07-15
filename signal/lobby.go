package main

import "sync"

type Lobby struct {
	membersMu sync.Mutex
	members   map[connID]*conn
}

func NewLobby() *Lobby {
	return &Lobby{
		members: map[connID]*conn{},
	}
}

func (l *Lobby) Add(c *conn) {
	l.membersMu.Lock()
	l.members[c.ID] = c
	l.membersMu.Unlock()
}

func (l *Lobby) Pop() []*conn {
	var members []*conn

	l.membersMu.Lock()
	for _, c := range l.members {
		members = append(members, c)
	}
	l.members = map[connID]*conn{}
	l.membersMu.Unlock()

	return members
}
