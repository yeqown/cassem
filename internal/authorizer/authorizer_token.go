package authorizer

import (
	"fmt"
	"strconv"

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
	UserId int
}

func NewToken(uid int) *Token {
	return &Token{UserId: uid}
}

func (t Token) Subject() string {
	return "uid:" + strconv.Itoa(t.UserId)
}

func Session(tokenString string) (*Token, error) {
	uid, err := parseToken(tokenString)
	if err != nil {
		return nil, err
	}

	return NewToken(uid), nil
}

// DONE(@yeqown): extract secret from code to global.
func genToken(u *persistence.User) (string, error) {
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
