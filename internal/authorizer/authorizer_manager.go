package authorizer

import (
	"fmt"

	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/pkg/hash"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

var (
	_SECRET_ = []byte("cassem")
)

func (c casbinAuthorities) AddUser(account, password, name string) error {
	return c.userRepo.Create(&persistence.UserDO{
		Account:          account,
		PasswordWithSalt: hash.WithSalt(password, "cassem"),
		Name:             name,
	})
}

func (c casbinAuthorities) Login(account, password string) (string, error) {
	u, err := c.userRepo.QueryUser(account)
	if err != nil {
		return "", err
	}

	if u.PasswordWithSalt != hash.WithSalt(password, "cassem") {
		return "", errors.New("account and password could not match")
	}

	// DONE(@yeqown): generate jwt token
	return genToken(u)
}

func (c casbinAuthorities) Session(tokenString string) (*Token, error) {
	uid, err := parseToken(tokenString)
	if err != nil {
		return nil, err
	}

	return &Token{
		UserId: uid,
	}, nil
}

// DONE(@yeqown): extract secret from code to global.
func genToken(u *persistence.UserDO) (string, error) {
	atClaims := jwt.MapClaims{}
	atClaims["authorized"] = true
	atClaims["user_id"] = u.ID
	//atClaims["exp"] = time.Now().Add(time.Minute * 15).Unix()
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	token, err := at.SignedString(_SECRET_)
	if err != nil {
		return "", errors.Wrap(err, "sign token failed")
	}

	return token, nil
}

func parseToken(tokenString string) (int, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Make sure that the token method conform to "SigningMethodHMAC"
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return _SECRET_, nil
	})
	if err != nil {
		return 0, err
	}

	at := token.Claims.(jwt.MapClaims)
	uid, ok := at["user_id"].(float64)
	log.
		WithFields(log.Fields{
			"at":      at,
			"user_id": at["user_id"],
		}).
		Debugf("parseToken with claims: %T", at["user_id"])
	if !ok {
		return 0, errors.New("parseToken could not convert user_id into int")
	}

	return int(uid), nil
}

func (c casbinAuthorities) ResetPassword(account, password string) error {
	return c.userRepo.ResetPassword(account, hash.WithSalt(password, "cassem"))
}
