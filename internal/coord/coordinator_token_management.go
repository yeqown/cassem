package coord

import (
	"context"
	"errors"
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/yeqown/cassem/internal/concept"
)

// Token management checks persisted permissions as well as the HTTP route's
// enforcer, which may have loaded its policy on another admin node earlier.
func (s *tokenService) authorizeManagement(ctx context.Context, account, action string) error {
	operator := concept.OperatorFromContext(ctx)
	if !validTokenAccount(operator) {
		return concept.Err_UNAUTHENTICATED
	}
	if err := s.validateUser(ctx, operator); err != nil {
		if errors.Is(err, concept.Err_NOT_FOUND) {
			return concept.Err_UNAUTHENTICATED
		}
		return fmt.Errorf("token management operator: %w", err)
	}
	e, err := s.currentPolicy(ctx)
	if err != nil {
		return err
	}
	allow, err := e.Enforce(operator, concept.Domain_CLUSTER, concept.Object_ACL, action)
	if err != nil {
		return fmt.Errorf("token management policy: %w", concept.Err_INTERNAL)
	}
	if !allow {
		return concept.Err_PERMISSION_DENIED
	}
	if action != concept.Action_WRITE {
		return nil
	}
	protected, err := hasSuperadminBinding(e, account, "")
	if err != nil || !protected {
		return err
	}
	globalSuperadmin, err := hasSuperadminBinding(e, operator, concept.Domain_ALL)
	if err != nil {
		return err
	}
	if !globalSuperadmin {
		return concept.Err_PERMISSION_DENIED
	}
	return nil
}

func hasSuperadminBinding(e *casbin.Enforcer, account, domain string) (bool, error) {
	bindings, err := e.GetFilteredGroupingPolicy(0, account)
	if err != nil {
		return false, fmt.Errorf("token management roles: %w", concept.Err_INTERNAL)
	}
	for _, binding := range bindings {
		if len(binding) >= 3 && binding[1] == rbacRole(concept.Role_SUPERADMIN) && (domain == "" || binding[2] == domain) {
			return true, nil
		}
	}
	return false, nil
}
