-- name: UsersServiceVerifyUserPasswordGetHash :one
SELECT password FROM users WHERE id = @id;
