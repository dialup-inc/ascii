package main

import "sync"

type Group struct {
	membersMu sync.Mutex
	members   map[connID]*conn
}

func NewGroup() *Group {
	return &Group{
		members: map[connID]*conn{},
	}
}

func (l *Group) Add(c *conn) {
	l.membersMu.Lock()
	l.members[c.ID] = c
	l.membersMu.Unlock()
}

func (l *Group) Remove(id connID) *conn {
	l.membersMu.Lock()
	defer l.membersMu.Unlock()

	c, exist := l.members[id]
	if !exist {
		return nil
	}

	delete(l.members, id)

	return c
}

func (l *Group) Pop() []*conn {
	var members []*conn

	l.membersMu.Lock()
	for _, c := range l.members {
		members = append(members, c)
	}
	l.members = map[connID]*conn{}
	l.membersMu.Unlock()

	return members
}
