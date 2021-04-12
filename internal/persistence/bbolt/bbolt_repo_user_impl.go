package bbolt

import (
	"encoding/json"
	"sync"

	"github.com/yeqown/cassem/internal/persistence"

	"github.com/casbin/casbin/v2/persist"
	"github.com/yeqown/log"
	bolt "go.etcd.io/bbolt"
)

var (
	_userBucket   = []byte("users")
	_policyBucket = []byte("users_policy")

	_once sync.Once
)

func (b bboltRepoImpl) CreateUser(u *persistence.User) error {
	do := &userDO{
		Account:  u.Account,
		Password: u.PasswordWithSalt,
		Name:     u.Name,
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		_once.Do(func() {
			if _, err := tx.CreateBucketIfNotExists(_userBucket); err != nil {
				log.
					Errorf("bboltRepoImpl.CreateUser could not create bucket")
			}
			// What should do if failed to create at first time ?
		})

		return tx.Bucket(_userBucket).Put(do.key(), do.value())
	})
}

func (b bboltRepoImpl) ResetPassword(account, passwordWithSalt string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		do := &userDO{
			Account: account,
		}

		v := tx.Bucket(_userBucket).Get(do.key())
		if v == nil {
			return ErrNotFound
		}

		if err := json.Unmarshal(v, do); err != nil {
			return err
		}

		do.Password = passwordWithSalt
		return tx.Bucket(_userBucket).Put(do.key(), do.value())
	})
}

func (b bboltRepoImpl) QueryUser(account string) (*persistence.User, error) {
	do := &userDO{
		Account: account,
	}
	err := b.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(_userBucket).Get(do.key())
		if v == nil {
			return ErrNotFound
		}

		return json.Unmarshal(v, do)
	})
	if err != nil {
		return nil, err
	}

	return &persistence.User{
		Account:          do.Account,
		PasswordWithSalt: do.Password,
		Name:             do.Name,
	}, err
}

func (b bboltRepoImpl) PagingUsers(filter *persistence.PagingUsersFilter) ([]*persistence.User, int, error) {
	out := make([]*persistence.User, 0, filter.Limit)

	err := b.db.View(func(tx *bolt.Tx) error {
		bu := tx.Bucket(_userBucket)

		var kvs []kv
		if filter.AccountPattern == "" {
			kvs, _ = pagingHelper(bu, filter.Offset, filter.Limit)
		} else {
			kvs, _ = pagingHelperWithPrefix(bu, filter.AccountPattern, filter.Offset, filter.Limit)
		}

		for _, item := range kvs {
			p := new(userDO)
			if err2 := json.Unmarshal(item.value, p); err2 != nil {
				log.
					WithFields(log.Fields{
						"value":     item.value,
						"bucketKey": item.key,
						"error":     err2,
					}).
					Warn("bboltRepoImpl.PagingContainers could not unmarshal container")

				continue
			}
			out = append(out, &persistence.User{
				Account:          p.Account,
				PasswordWithSalt: p.Password,
				Name:             p.Name,
			})
		}

		return nil
	})

	return out, 0, err
}

func (b bboltRepoImpl) PolicyAdapter() (persist.Adapter, error) {
	return newAdapter(b.db, _policyBucket), nil
}
