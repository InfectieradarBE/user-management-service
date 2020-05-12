package service

import (
	"context"
	"log"

	"github.com/influenzanet/user-management-service/pkg/api"
	"github.com/influenzanet/user-management-service/pkg/models"
	"github.com/influenzanet/user-management-service/pkg/pwhash"
	"github.com/influenzanet/user-management-service/pkg/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *userManagementServer) CreateUser(ctx context.Context, req *api.CreateUserReq) (*api.User, error) {
	if req == nil || utils.IsTokenEmpty(req.Token) || req.AccountId == "" || req.InitialPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "missing arguments")
	}
	if !utils.CheckRoleInToken(req.Token, "ADMIN") {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}

	if !utils.CheckEmailFormat(req.AccountId) {
		return nil, status.Error(codes.InvalidArgument, "account id not a valid email")
	}
	if !utils.CheckPasswordFormat(req.InitialPassword) {
		return nil, status.Error(codes.InvalidArgument, "password too weak")
	}

	password, err := pwhash.HashPassword(req.InitialPassword)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Create user DB object from request:
	newUser := models.User{
		Account: models.Account{
			Type:               "email",
			AccountID:          req.AccountId,
			AccountConfirmedAt: 0, // not confirmed yet
			Password:           password,
			PreferredLanguage:  req.PreferredLanguage,
		},
		Roles: req.Roles,
		Profiles: []models.Profile{
			{
				ID:       primitive.NewObjectID(),
				Alias:    req.AccountId,
				AvatarID: "default",
			},
		},
	}
	newUser.AddNewEmail(req.AccountId, false)

	instanceID := req.Token.InstanceId
	id, err := s.userDBservice.AddUser(instanceID, newUser)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	newUser.ID, _ = primitive.ObjectIDFromHex(id)

	log.Println("TODO: generate account confirmation token for newly created user")
	log.Println("TODO: send email for newly created user")
	// TODO: generate email confirmation token
	// TODO: send email with confirmation request

	return newUser.ToAPI(), nil
}

func (s *userManagementServer) AddRoleForUser(ctx context.Context, req *api.RoleMsg) (*api.User, error) {
	if req == nil || utils.IsTokenEmpty(req.Token) || req.AccountId == "" || req.Role == "" {
		return nil, status.Error(codes.InvalidArgument, "missing arguments")
	}
	if !utils.CheckRoleInToken(req.Token, "ADMIN") {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (s *userManagementServer) RemoveRoleForUser(ctx context.Context, req *api.RoleMsg) (*api.User, error) {
	if req == nil || utils.IsTokenEmpty(req.Token) || req.AccountId == "" || req.Role == "" {
		return nil, status.Error(codes.InvalidArgument, "missing arguments")
	}
	if !utils.CheckRoleInToken(req.Token, "ADMIN") {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (s *userManagementServer) FindNonParticipantUsers(ctx context.Context, req *api.FindNonParticipantUsersMsg) (*api.UserListMsg, error) {
	if req == nil || utils.IsTokenEmpty(req.Token) {
		return nil, status.Error(codes.InvalidArgument, "missing arguments")
	}
	if !utils.CheckRoleInToken(req.Token, "ADMIN") {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}