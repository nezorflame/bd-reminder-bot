package main

import (
	"bytes"
	"time"

	bolt "github.com/coreos/bbolt"
	"github.com/pkg/errors"
)

// DB is a cache, wraps bolt.DB
type DB struct {
	ManagerBucketName []byte
	ChannelBucketName []byte

	*bolt.DB
}

// Default DB consts
const (
	DefaultDBTimeout = 1 * time.Second
	DefaultDBExpTime = 240 * time.Hour
)

func openDB(path *string, mBucket, cBucket string, timeout time.Duration) (*DB, error) {
	if timeout == 0 {
		timeout = DefaultDBTimeout
	}

	// open DB
	boltDB, err := bolt.Open(*path, 0600, &bolt.Options{Timeout: timeout})
	if err != nil {
		return nil, err
	}
	db := &DB{[]byte(mBucket), []byte(cBucket), boltDB}

	// create buckets if needed
	if err = db.newBucket(db.ManagerBucketName); err != nil {
		return nil, err
	}
	if err = db.newBucket(db.ChannelBucketName); err != nil {
		return nil, err
	}

	return db, nil
}

// SaveUserBDToCache saves the record about the user's birthday into the cache DB's bucket
func (db *DB) SaveUserBDToCache(bucketName []byte, id, bd string) error {
	if !bytes.Equal(bucketName, db.ManagerBucketName) && !bytes.Equal(bucketName, db.ChannelBucketName) {
		return errors.Errorf("bucket %q does not exist", bucketName)
	}

	if err := db.put(bucketName, []byte(id), []byte(bd)); err != nil {
		return errors.Wrap(err, "unable to put value into DB")
	}

	return nil
}

// CheckUserBDInCache checks if the record about the user's birthday is present in the cache DB's bucket
func (db *DB) CheckUserBDInCache(bucketName []byte, id, bd string) (bool, error) {
	if !bytes.Equal(bucketName, db.ManagerBucketName) && !bytes.Equal(bucketName, db.ChannelBucketName) {
		return false, errors.Errorf("bucket %q does not exist", bucketName)
	}

	dbValue, err := db.get(bucketName, []byte(id))
	if err != nil {
		return false, errors.Wrap(err, "unable to get value from DB")
	}

	if dbValue != nil {
		// check if stored BD record is the same
		if bytes.Equal([]byte(bd), dbValue) {
			return true, nil
		}
	}

	return false, nil
}

func (db *DB) newBucket(bucketName []byte) error {
	return db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketName)
		return err
	})
}

func (db *DB) put(bucketName, key, value []byte) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		if bucket == nil {
			return errors.Errorf("bucket %q not found", bucketName)
		}

		return bucket.Put(key, value)
	})
}

func (db *DB) get(bucketName, key []byte) (value []byte, err error) {
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		if bucket == nil {
			return errors.Errorf("bucket %q not found", bucketName)
		}

		if v := bucket.Get(key); v != nil {
			// copy v into value as v only lives till the end of tx
			value = make([]byte, len(v))
			copy(value, v)
		}

		return nil
	})
	return
}
