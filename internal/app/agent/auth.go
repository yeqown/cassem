package agent

import (
	"context"
	"strings"

	"github.com/yeqown/cassem/internal/concept"
	"google.golang.org/grpc/metadata"
)

func bearerToken(ctx context.Context) (string, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	values := md.Get("authorization")
	if len(values) != 1 {
		return "", concept.Err_UNAUTHENTICATED
	}
	scheme, token, ok := strings.Cut(values[0], " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", concept.Err_UNAUTHENTICATED
	}
	for _, c := range token {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', strings.ContainsRune("-._~+/=", c):
		default:
			return "", concept.Err_UNAUTHENTICATED
		}
	}
	return token, nil
}

func (d *app) authorize(ctx context.Context, targets ...concept.TokenTarget) error {
	token, err := bearerToken(ctx)
	if err != nil {
		return err
	}
	if d.authorizer == nil {
		return concept.Err_UNAUTHENTICATED
	}
	return d.authorizer.AuthorizeToken(ctx, token, targets...)
}
