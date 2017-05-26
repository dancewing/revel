package orm

import (
	"sync"
)

var (
	database *databaseSingleton
	once     sync.Once
)

type databaseSingleton struct {
	dbmap *DbMap
	mu    sync.RWMutex
}

func (r *databaseSingleton) Set(dbmap *DbMap) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dbmap = dbmap
}

func (r *databaseSingleton) Get() *DbMap {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.dbmap == nil {
		panic("DbMap is no set")
	}
	return r.dbmap
}

func Database() *databaseSingleton {
	if database == nil {
		database = &databaseSingleton{}
	}
	return database
}
