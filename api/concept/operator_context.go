package concept

import "context"

type operatorContextKey struct{}

func WithOperator(ctx context.Context, operator string) context.Context {
	return context.WithValue(ctx, operatorContextKey{}, operator)
}

func OperatorFromContext(ctx context.Context) string {
	operator, ok := ctx.Value(operatorContextKey{}).(string)
	if !ok || operator == "" {
		return "system"
	}
	return operator
}
