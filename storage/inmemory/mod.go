package inmemory

import (
	"sync"

	"go.dedis.ch/cs438/storage"
)

// NewPersistency return a new initialized in-memory storage. Opeartions are
// thread-safe with a global mutex.
func NewPersistency() storage.Storage {
	return Storage{
		blob:       newStore(),
		naming:     newStore(),
		blockchain: newStore(),
	}
}

// Storage implements an in-memory storage.
//
// - implements storage.Storage
type Storage struct {
	blob       storage.Store
	naming     storage.Store
	blockchain storage.Store
}

// GetDataBlobStore implements storage.Storage
func (s Storage) GetDataBlobStore() storage.Store {
	return s.blob
}

// GetNamingStore implements storage.Storage
func (s Storage) GetNamingStore() storage.Store {
	return s.naming
}

// GetBlockchainStore implements storage.Storage
func (s Storage) GetBlockchainStore() storage.Store {
	return s.blockchain
}

func newStore() *store {
	return &store{
		data: make(map[string][]byte),
	}
}

// store implements an in-memory store.
//
// - implements storage.Store
type store struct {
	sync.RWMutex
	data map[string][]byte
}

// Get implements storage.Store
func (s *store) Get(key string) (val []byte) {
	s.Lock()
	defer s.Unlock()

	return s.data[string(key)]
}

// Set implements storage.Store
func (s *store) Set(key string, val []byte) {
	s.Lock()
	defer s.Unlock()

	s.data[string(key)] = val
}

// Delete implements storage.Store
func (s *store) Delete(key string) {
	s.Lock()
	defer s.Unlock()

	delete(s.data, string(key))
}

// ForEach implements storage.Store
func (s *store) ForEach(f func(key string, val []byte) bool) {
	s.Lock()
	defer s.Unlock()

	for k, v := range s.data {
		cont := f(k, v)
		if !cont {
			return
		}
	}
}

// Len implements storage.Store
func (s *store) Len() int {
	s.Lock()
	defer s.Unlock()

	return len(s.data)
}

// Get all data from store
func (s *store) GetAll() map[string][]byte {
	s.RLock()
	defer s.RUnlock()
	ret := map[string][]byte{}
	for k, v := range s.data {
		tmp := make([]byte, len(v))
		copy(tmp, v)
		if k == storage.LastBlockKey {
			ret["LastBlockKey"] = v
		} else {
			ret[k] = v
		}
	}
	return ret
}

//------------------------- BlockChain for HR ------------------------------------

// NewPersistency return a new initialized in-memory storage. Opeartions are
// thread-safe with a global mutex.
func NewHrPersistency() storage.HrStorage {
	return HrStorage{
		hrNaming:     newHrStore(),
		hrBlockchain: newHrStore(),
	}
}

// HrStorage implements an in-memory storage.
//
// - implements storage.HrStorage
type HrStorage struct {
	hrNaming     storage.HrStore
	hrBlockchain storage.HrStore
}

// GetHrNamingStore implements storage.Storage
func (hs HrStorage) GetHrNamingStore() storage.HrStore {
	return hs.hrNaming
}

// GetHrBlockchainStore implements storage.HrStorage
func (hs HrStorage) GetHrBlockchainStore() storage.HrStore {
	return hs.hrBlockchain
}

func newHrStore() *hrStore {
	return &hrStore{
		data: make(map[string]map[string][]byte),
	}
}

// store implements an in-memory store.
//
// - implements storage.HrStore
type hrStore struct {
	sync.RWMutex
	data map[string]map[string][]byte
}

// Get implements storage.HrStore
func (hs *hrStore) Get(IDhr string, key string) (val []byte) {
	hs.Lock()
	defer hs.Unlock()

	_, ok := hs.data[IDhr]
	if !ok {
		hs.data[IDhr] = make(map[string][]byte)
		return nil
	}
	block, ok := hs.data[IDhr][string(key)]
	if !ok {
		return nil
	}
	return block
}

// Set implements storage.HrStore
func (hs *hrStore) Set(IDhr string, key string, val []byte) {
	hs.Lock()
	defer hs.Unlock()

	_, ok := hs.data[IDhr]
	if !ok {
		hs.data[IDhr] = make(map[string][]byte)
	}

	hs.data[IDhr][string(key)] = val
}

// Delete implements storage.HrStore
func (hs *hrStore) Delete(IDhr string, key string) {
	hs.Lock()
	defer hs.Unlock()

	_, ok := hs.data[IDhr]
	if !ok {
		hs.data[IDhr] = make(map[string][]byte)
	}

	delete(hs.data[IDhr], string(key))
}

// Len implements storage.HrStore
func (hs *hrStore) Len(IDhr string) int {
	hs.Lock()
	defer hs.Unlock()

	_, ok := hs.data[IDhr]
	if !ok {
		hs.data[IDhr] = make(map[string][]byte)
	}

	return len(hs.data[IDhr])
}

// Get all data from HrStore
func (hs *hrStore) GetAll(IDhr string) map[string][]byte {
	hs.RLock()
	defer hs.RUnlock()
	ret := map[string][]byte{}

	_, ok := hs.data[IDhr]
	if !ok {
		hs.data[IDhr] = make(map[string][]byte)
	}

	for k, v := range hs.data[IDhr] {
		tmp := make([]byte, len(v))
		copy(tmp, v)
		if k == storage.LastBlockKey {
			ret["LastBlockKey"] = v
		} else {
			ret[k] = v
		}
	}
	return ret
}
