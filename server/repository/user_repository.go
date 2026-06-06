package repository

import (
	"go.woodpecker-ci.org/woodpecker/v3/server/model"
)

// UserRepository defines the interface for user data access operations
type UserRepository interface {
	// GetUser gets a user by unique ID
	GetUser(int64) (*model.User, error)

	// GetUserByRemoteID gets a user by remote ID
	GetUserByRemoteID(int64, model.ForgeRemoteID) (*model.User, error)

	// GetUserByLogin gets a user by its login name
	GetUserByLogin(int64, string) (*model.User, error)

	// GetUserList gets a list of all users in the system
	GetUserList(*model.ListOptions) ([]*model.User, error)

	// GetUserCount gets a count of all users in the system
	GetUserCount() (int64, error)

	// CreateUser creates a new user account
	CreateUser(*model.User) error

	// UpdateUser updates a user account
	UpdateUser(*model.User) error

	// DeleteUser deletes a user account
	DeleteUser(*model.User) error
}
