// Package badgerkv provides an adapter to Badger's key-value store that is
// compatible with libkv's Store interface.
package badgerkv

import (
	"bytes"

	"github.com/dgraph-io/badger"
	"github.com/docker/libkv/store"
)

const (
	MaxValueLogFileSize      = 256 << 20
	MaxDeleteTransactionSize = 1024
)

type Store struct {
	db *badger.DB
}

func NewStore(name string) (*Store, error) {
	opts := badger.DefaultOptions
	opts.Dir = name
	opts.ValueDir = name
	opts.ValueLogFileSize = MaxValueLogFileSize
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (t *Store) DB() *badger.DB {
	return t.db
}

func (t *Store) Close() {
	t.db.Close()
}

func (t *Store) Get(key string) (*store.KVPair, error) {
	return t.GetInto(key, nil)
}

func (t *Store) GetInto(key string, buf []byte) (*store.KVPair, error) {
	var val []byte
	err := t.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		val, err = item.ValueCopy(buf)
		return err
	})
	if err == badger.ErrKeyNotFound {
		return nil, store.ErrKeyNotFound
	} else if err != nil {
		return nil, err
	}
	return &store.KVPair{Key: key, Value: val}, nil
}

func (t *Store) Exists(key string) (bool, error) {
	err := t.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		return err
	})
	if err == badger.ErrKeyNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (t *Store) Put(key string, value []byte, options *store.WriteOptions) error {
	return t.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
}

func (t *Store) Delete(key string) error {
	return t.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// DeleteRange deletes the range of keys [start, end). The range is open and
// the end key is not deleted. Returns nil if all keys in the range are
// deleted. There are no guarantees which keys have been deleted on error.
func (t *Store) DeleteRange(start, end string) error {
	startKey := []byte(start)
	endKey := []byte(end)
	more := true
	var err error
	for more && err == nil {
		more = false
		err = t.db.Update(func(txn *badger.Txn) error {
			iter := txn.NewIterator(badger.IteratorOptions{})
			defer iter.Close()
			iter.Seek(startKey)
			for i := 0; iter.Valid() && i < MaxDeleteTransactionSize; i++ {
				if bytes.Compare(iter.Item().Key(), endKey) >= 0 {
					break
				}
				// Txn.Delete holds onto the key slice, so we have to make a copy
				// before passing. Sigh!
				err := txn.Delete(append([]byte(nil), iter.Item().Key()...))
				if err != nil {
					return err
				}
				more = true
				iter.Next()
			}
			return nil
		})
	}
	return err
}

func (t *Store) AtomicPut(key string, value []byte, previous *store.KVPair, options *store.WriteOptions) (bool, *store.KVPair, error) {
	bKey := []byte(key)

	err := t.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(bKey)
		if err == badger.ErrKeyNotFound {
			if previous != nil {
				return store.ErrKeyNotFound
			}
		} else if err != nil {
			return err
		} else if previous == nil {
			return store.ErrKeyExists
		}

		if previous != nil {
			oldVal, err := item.Value()
			if err != nil {
				return err
			}
			if !bytes.Equal(previous.Value, oldVal) {
				return store.ErrKeyModified
			}
		}

		return txn.Set(bKey, value)
	})

	if err != nil {
		return false, nil, err
	}

	updated := &store.KVPair{
		Key:   key,
		Value: value,
	}
	return true, updated, nil
}

func (t *Store) AtomicDelete(key string, previous *store.KVPair) (bool, error) {
	if previous == nil {
		return false, store.ErrPreviousNotSpecified
	}

	bKey := []byte(key)
	err := t.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(bKey)
		if err == badger.ErrKeyNotFound {
			return store.ErrKeyNotFound
		} else if err != nil {
			return err
		}

		oldVal, err := item.Value()
		if err != nil {
			return err
		}
		if previous != nil && !bytes.Equal(previous.Value, oldVal) {
			return store.ErrKeyModified
		}

		return txn.Delete(bKey)
	})

	return err == nil, err
}

func (*Store) Watch(key string, stopCh <-chan struct{}) (<-chan *store.KVPair, error) {
	return nil, store.ErrCallNotSupported
}

func (*Store) WatchTree(directory string, stopCh <-chan struct{}) (<-chan []*store.KVPair, error) {
	return nil, store.ErrCallNotSupported
}

func (*Store) NewLock(key string, options *store.LockOptions) (store.Locker, error) {
	return nil, store.ErrCallNotSupported
}

func (*Store) List(directory string) ([]*store.KVPair, error) {
	return nil, store.ErrCallNotSupported
}

func (*Store) DeleteTree(directory string) error {
	return store.ErrCallNotSupported
}
