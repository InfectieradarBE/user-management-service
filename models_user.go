package main

import (
	"errors"

	api "github.com/influenzanet/user-management-service/api"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ObjectInfos describes metadata for the User
type ObjectInfos struct {
	LastTokenRefresh int64 `bson:"lastTokenRefresh"`
	LastLogin        int64 `bson:"lastLogin"`
	CreatedAt        int64 `bson:"createdAt"`
	UpdatedAt        int64 `bson:"updatedAt"`
}

// ToAPI converts the object from DB to API format
func (o ObjectInfos) ToAPI() *api.User_Infos {
	return &api.User_Infos{
		LastTokenRefresh: o.LastTokenRefresh,
		LastLogin:        o.LastLogin,
		CreatedAt:        o.CreatedAt,
		UpdatedAt:        o.UpdatedAt,
	}
}

// User describes the user as saved in the DB
type User struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"user_id,omitempty"`
	Account     Account            `bson:"account"`
	Roles       []string           `bson:"roles" json:"roles"`
	ObjectInfos ObjectInfos        `bson:"objectInfos"`
	Profiles    []Profile          `bson:"profiles"`
}

// ToAPI converts the object from DB to API format
func (u User) ToAPI() *api.User {
	return &api.User{
		Id:      u.ID.Hex(),
		Account: u.Account.ToAPI(),
		Roles:   u.Roles,
		/*
			Profile:     u.Profile.ToAPI(),
			SubProfiles: u.SubProfiles.ToAPI(),
		*/
		Infos: u.ObjectInfos.ToAPI(),
	}
}

// HasRole checks whether the user has a specified role
func (u User) HasRole(role string) bool {
	for _, v := range u.Roles {
		if v == role {
			return true
		}
	}
	return false
}

/*
// AddSubProfile generates unique ID and adds sub-profile to the user's array
func (u *User) AddSubProfile(sp SubProfile) {
	sp.ID = primitive.NewObjectID()
	u.SubProfiles = append(u.SubProfiles, sp)
}

// UpdateSubProfile finds and replaces sub-profile in the user's array
func (u *User) UpdateSubProfile(sp SubProfile) error {
	for i, cSP := range u.SubProfiles {
		if cSP.ID == sp.ID {
			u.SubProfiles[i] = sp
			return nil
		}
	}
	return errors.New("item with given ID not found")
}

// RemoveSubProfile finds and removes sub-profile from the user's array
func (u *User) RemoveSubProfile(id string) error {
	for i, cSP := range u.SubProfiles {
		if cSP.ID.Hex() == id {
			u.SubProfiles = append(u.SubProfiles[:i], u.SubProfiles[i+1:]...)
			return nil
		}
	}
	return errors.New("item with given ID not found")
}
*/

// AddRefreshToken add a new refresh token to the user's list
func (u *User) AddRefreshToken(token string) {
	u.Account.RefreshTokens = append(u.Account.RefreshTokens, token)
	if len(u.Account.RefreshTokens) > 10 {
		_, u.Account.RefreshTokens = u.Account.RefreshTokens[0], u.Account.RefreshTokens[1:]
	}
}

// HasRefreshToken checks weather a user has a particular refresh token
func (u *User) HasRefreshToken(token string) bool {
	for _, t := range u.Account.RefreshTokens {
		if t == token {
			return true
		}
	}
	return false
}

// RemoveRefreshToken deletes a refresh token from the user's list
func (u *User) RemoveRefreshToken(token string) error {
	for i, t := range u.Account.RefreshTokens {
		if t == token {
			u.Account.RefreshTokens = append(u.Account.RefreshTokens[:i], u.Account.RefreshTokens[i+1:]...)
			return nil
		}
	}
	return errors.New("token was missing")
}
