package users

import (
	"context"

	"github.com/eduardolat/pgbackweb/internal/util/cryptoutil"
	"github.com/google/uuid"
)

// VerifyUserPassword returns an error when password is not the current
// password of the user.
func (s *Service) VerifyUserPassword(
	ctx context.Context, userID uuid.UUID, password string,
) error {
	hash, err := s.dbgen.UsersServiceVerifyUserPasswordGetHash(ctx, userID)
	if err != nil {
		return err
	}
	return cryptoutil.VerifyBcryptHash(password, hash)
}
