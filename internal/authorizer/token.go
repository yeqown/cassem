package authorizer

import (
	"fmt"

	"github.com/yeqown/cassem/internal/persistence"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

var (
	_SECRET_ = []byte("cassem")
)

// Token is the bridge between authorizer and HTTP API.
type Token struct {
	Account string
}

func NewToken(account string) *Token {
	return &Token{Account: account}
}

func (t Token) Subject() string {
	return "uid:" + t.Account
}

func Session(tokenString string) (*Token, error) {
	account, err := parseToken(tokenString)
	if err != nil {
		return nil, err
	}

	return NewToken(account), nil
}

// DONE(@yeqown): extract secret from code to global.
func GenToken(u *persistence.User) (string, error) {
	atClaims := jwt.MapClaims{}
	atClaims["authorized"] = true
	atClaims["account"] = u.Account
	//atClaims["exp"] = time.Now().Add(time.Minute * 15).Unix()
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	token, err := at.SignedString(_SECRET_)
	if err != nil {
		return "", errors.Wrap(err, "sign token failed")
	}

	return token, nil
}

func parseToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Make sure that the token method conform to "SigningMethodHMAC"
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return _SECRET_, nil
	})
	if err != nil {
		return "", err
	}

	at := token.Claims.(jwt.MapClaims)
	account, ok := at["account"].(string)
	log.
		WithFields(log.Fields{
			"at":      at,
			"account": at["account"],
		}).
		Debugf("parseToken with claims: %T", at["account"])
	if !ok {
		return "", errors.New("parseToken could not convert user_id into int")
	}

	return account, nil
}
