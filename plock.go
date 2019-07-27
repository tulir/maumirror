// maumirror - A GitHub repo mirroring system using webhooks.
// Copyright (C) 2019 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

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
