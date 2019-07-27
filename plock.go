// From https://www.reddit.com/r/golang/comments/6w37q3/-/dm84azo

package main

import (
	"sync"
)

type PartitionLocker struct {
	cond *sync.Cond
	lock sync.Locker
	part map[string]struct{}
}

func NewPartitionLocker(lock sync.Locker) *PartitionLocker {
	return &PartitionLocker{
		cond: sync.NewCond(lock),
		lock: lock,
		part: make(map[string]struct{}),
	}
}

func (pl *PartitionLocker) locked(id string) (ok bool) { _, ok = pl.part[id]; return }

func (pl *PartitionLocker) Lock(id string) {
	pl.lock.Lock()
	defer pl.lock.Unlock()
	for pl.locked(id) {
		pl.cond.Wait()
	}
	pl.part[id] = struct{}{}
	return
}

func (pl *PartitionLocker) Unlock(id string) {
	pl.lock.Lock()
	defer pl.lock.Unlock()
	delete(pl.part, id)
	pl.cond.Broadcast()
}
