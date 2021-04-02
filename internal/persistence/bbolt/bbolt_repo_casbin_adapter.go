package bbolt

import (
	"encoding/json"
	"errors"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	bolt "go.etcd.io/bbolt"
)

var (
	// errEmptyPolicy will be returned if the bucket doesn't have any policy data
	errEmptyPolicy = errors.New("policy was empty")
)

// policyRule represents a policy type and their values
type policyRule struct {
	PType string
	V0    string
	V1    string
	V2    string
	V3    string
	V4    string
	V5    string
}

// casbinBoltAdapter represents the BoltDB casbinBoltAdapter for policy storage.
type casbinBoltAdapter struct {
	db        *bolt.DB
	bucketKey []byte
}

// NewBoltAdapter is the constructor for casbinBoltAdapter. Assumes the bolt db is already opened.
func newAdapter(db *bolt.DB, bucketKey []byte) persist.Adapter {
	a := &casbinBoltAdapter{}
	a.db = db
	a.bucketKey = bucketKey

	a.open()

	return a
}

func (a *casbinBoltAdapter) open() {
	err := a.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(a.bucketKey)
		return err
	})

	// i don't like panic'ing here but that's what the other adapters do.
	if err != nil {
		panic(err)
	}
}

func loadPolicyLine(line policyRule, model model.Model) {
	lineText := line.PType
	if line.V0 != "" {
		lineText += ", " + line.V0
	}
	if line.V1 != "" {
		lineText += ", " + line.V1
	}
	if line.V2 != "" {
		lineText += ", " + line.V2
	}
	if line.V3 != "" {
		lineText += ", " + line.V3
	}
	if line.V4 != "" {
		lineText += ", " + line.V4
	}
	if line.V5 != "" {
		lineText += ", " + line.V5
	}

	persist.LoadPolicyLine(lineText, model)
}

// LoadPolicy loads policy from database.
func (a *casbinBoltAdapter) LoadPolicy(model model.Model) error {
	return a.db.View(func(tx *bolt.Tx) error {
		lines := make([]policyRule, 0)
		bucket := tx.Bucket(a.bucketKey)
		policy := bucket.Get([]byte("policy"))
		if policy == nil {
			return errEmptyPolicy
		}

		if err := json.Unmarshal(policy, &lines); err != nil {
			return err
		}

		for _, line := range lines {
			loadPolicyLine(line, model)
		}

		return nil
	})
}

func savePolicyLine(ptype string, rule []string) policyRule {
	line := policyRule{}

	line.PType = ptype
	if len(rule) > 0 {
		line.V0 = rule[0]
	}
	if len(rule) > 1 {
		line.V1 = rule[1]
	}
	if len(rule) > 2 {
		line.V2 = rule[2]
	}
	if len(rule) > 3 {
		line.V3 = rule[3]
	}
	if len(rule) > 4 {
		line.V4 = rule[4]
	}
	if len(rule) > 5 {
		line.V5 = rule[5]
	}

	return line
}

// SavePolicy saves policy to database.
func (a *casbinBoltAdapter) SavePolicy(model model.Model) error {
	var rules []policyRule
	for ptype, ast := range model["p"] {
		for _, rule := range ast.Policy {
			line := savePolicyLine(ptype, rule)
			rules = append(rules, line)
		}
	}

	for ptype, ast := range model["g"] {
		for _, rule := range ast.Policy {
			line := savePolicyLine(ptype, rule)
			rules = append(rules, line)
		}
	}

	text, err := json.Marshal(rules)
	if err != nil {
		return err
	}

	return a.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(a.bucketKey)
		return bucket.Put([]byte("policy"), text)
	})
}

// AddPolicy adds a policy rule to the storage. [auto save]
func (a *casbinBoltAdapter) AddPolicy(sec string, ptype string, rule []string) error {
	return errors.New("not implemented")
}

// RemovePolicy removes a policy rule from the storage. [auto save]
func (a *casbinBoltAdapter) RemovePolicy(sec string, ptype string, rule []string) error {
	return errors.New("not implemented")
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage. [auto save]
func (a *casbinBoltAdapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	return errors.New("not implemented")
}
