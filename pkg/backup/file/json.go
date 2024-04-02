package file

import (
	"encoding/json"
	"errors"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const BcryptRounds = 12

//nolint:wrapcheck //wrapping adds nothing and hurts the error message's readability
func (u *User) UnmarshalJSON(b []byte) error {
	type tmpType User

	var tmpJSON tmpType

	if err := json.Unmarshal(b, &tmpJSON); err != nil {
		return err
	}

	*u = User(tmpJSON)

	if u.PasswordHash == "" && u.Password != "" {
		hash, err := utils.HashPassword(BcryptRounds, u.Password)
		if err != nil {
			return err
		}

		u.PasswordHash = hash
	}

	return nil
}

//nolint:wrapcheck //wrapping adds nothing and hurts the error message's readability
func (l *LocalAgent) UnmarshalJSON(b []byte) error {
	type tmpType LocalAgent

	var tmpJSON tmpType

	if err := json.Unmarshal(b, &tmpJSON); err != nil {
		return err
	}

	*l = LocalAgent(tmpJSON)

	for i := range l.Accounts {
		acc := &l.Accounts[i]

		if acc.PasswordHash == "" && acc.Password != "" {
			if l.Protocol == "r66" || l.Protocol == "r66-tls" {
				acc.Password = utils.R66Hash(acc.Password)
			}

			var err error
			if acc.PasswordHash, err = utils.HashPassword(BcryptRounds, acc.Password); err != nil {
				return err
			}
		}
	}

	return nil
}

//nolint:wrapcheck //wrapping adds nothing and hurts the error message's readability
func (r *RemoteAgent) UnmarshalJSON(b []byte) error {
	type tmpType RemoteAgent

	var tmpJSON tmpType

	if err := json.Unmarshal(b, &tmpJSON); err != nil {
		return err
	}

	*r = RemoteAgent(tmpJSON)

	if r.Protocol != "r66" && r.Protocol != "r66-tls" {
		return nil // nothing to do
	}

	servPwdAny, hasPwd := r.Configuration["serverPassword"]
	if !hasPwd {
		return nil
	}

	servPwd, isStr := servPwdAny.(string)
	if !isStr {
		//nolint:goerr113 //too specific
		return errors.New(`the R66 "serverPassword" field must have type string`)
	}

	if !utils.IsHash(servPwd) {
		servPwd = utils.R66Hash(servPwd)

		hash, err := utils.HashPassword(BcryptRounds, servPwd)
		if err != nil {
			return err
		}

		r.Configuration["serverPassword"] = hash
	}

	return nil
}
