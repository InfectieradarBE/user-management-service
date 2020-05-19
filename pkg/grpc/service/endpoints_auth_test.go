package service

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/influenzanet/user-management-service/pkg/api"
	"github.com/influenzanet/user-management-service/pkg/models"
	"github.com/influenzanet/user-management-service/pkg/pwhash"
	"github.com/influenzanet/user-management-service/pkg/tokens"
	messageMock "github.com/influenzanet/user-management-service/test/mocks/messaging_service"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/status"
)

func TestLogin(t *testing.T) {
	s := userManagementServer{
		userDBservice:   testUserDBService,
		globalDBService: testGlobalDBService,
		JWT: models.JWTConfig{
			TokenMinimumAgeMin:  time.Second * 1,
			TokenExpiryInterval: time.Second * 2,
		},
	}

	// Create Test User
	currentPw := "SuperSecurePassword123!§$"
	hashedPw, err := pwhash.HashPassword(currentPw)
	if err != nil {
		t.Errorf("error creating user for testing login")
		return
	}

	testUser := models.User{
		Account: models.Account{
			Type:               "email",
			AccountID:          "test-login@test.com",
			AccountConfirmedAt: time.Now().Unix(),
			Password:           hashedPw,
			PreferredLanguage:  "de",
		},
		Roles: []string{"PARTICIPANT"},
		Profiles: []models.Profile{
			{ID: primitive.NewObjectID()},
		},
	}

	id, err := testUserDBService.AddUser(testInstanceID, testUser)
	if err != nil {
		t.Errorf("error creating user for testing login")
		return
	}
	testUser.ID, err = primitive.ObjectIDFromHex(id)
	if err != nil {
		t.Errorf("error converting id")
		return
	}

	t.Run("without payload", func(t *testing.T) {
		resp, err := s.LoginWithEmail(context.Background(), nil)
		st, ok := status.FromError(err)
		if !ok || st == nil || st.Message() != "invalid username and/or password" || resp != nil {
			t.Errorf("wrong error: %s", err.Error())
			t.Errorf("or response: %s", resp)
			return
		}
	})

	t.Run("with empty payload", func(t *testing.T) {
		req := &api.LoginWithEmailMsg{}

		resp, err := s.LoginWithEmail(context.Background(), req)
		if err == nil || status.Convert(err).Message() != "invalid username and/or password" || resp != nil {
			t.Errorf("wrong error: %s", err.Error())
			t.Errorf("or response: %s", resp)
			return
		}
	})

	t.Run("with wrong email", func(t *testing.T) {
		req := &api.LoginWithEmailMsg{
			Email:      "wrong@test.com",
			Password:   currentPw,
			InstanceId: testInstanceID,
		}

		resp, err := s.LoginWithEmail(context.Background(), req)
		if err == nil || resp != nil {
			t.Errorf("wrong response: %s", resp)
			return
		}
		if status.Convert(err).Message() != "invalid username and/or password" {
			t.Errorf("wrong error: %s", err.Error())
			return
		}
	})

	t.Run("with wrong password", func(t *testing.T) {
		req := &api.LoginWithEmailMsg{
			Email:      testUser.Account.AccountID,
			Password:   currentPw + "w",
			InstanceId: testInstanceID,
		}

		resp, err := s.LoginWithEmail(context.Background(), req)
		if err == nil || resp != nil {
			t.Errorf("wrong response: %s", resp)
			return
		}
		if status.Convert(err).Message() != "invalid username and/or password" {
			t.Errorf("wrong error: %s", err.Error())
			return
		}
	})

	t.Run("with valid fields", func(t *testing.T) {
		req := &api.LoginWithEmailMsg{
			Email:         testUser.Account.AccountID,
			Password:      currentPw,
			InstanceId:    testInstanceID,
			AsParticipant: true,
		}

		resp, err := s.LoginWithEmail(context.Background(), req)

		if err != nil {
			t.Errorf("unexpected error: %s", err.Error())
			return
		}
		if resp == nil || len(resp.AccessToken) < 1 || len(resp.RefreshToken) < 1 {
			t.Errorf("unexpected response: %s", resp)
			return
		}

		if resp.PreferredLanguage != "de" || resp.SelectedProfileId != testUser.Profiles[0].ID.Hex() {
			t.Errorf("unexpected PreferredLanguage or AccountConfirmed: %s", resp)
			return
		}
	})
}

func TestLoginWithTempToken(t *testing.T) {
	s := userManagementServer{
		userDBservice:   testUserDBService,
		globalDBService: testGlobalDBService,
		JWT: models.JWTConfig{
			TokenMinimumAgeMin:  time.Second * 1,
			TokenExpiryInterval: time.Second * 2,
		},
	}
	testUser := models.User{
		Account: models.Account{
			Type:               "email",
			AccountID:          "test-login-with-temptoken@test.com",
			AccountConfirmedAt: time.Now().Unix(),
			PreferredLanguage:  "de",
		},
		Roles: []string{"PARTICIPANT"},
		Profiles: []models.Profile{
			{
				ID:    primitive.NewObjectID(),
				Alias: "test",
			},
		},
	}

	id, err := testUserDBService.AddUser(testInstanceID, testUser)
	if err != nil {
		t.Errorf("error creating user for testing login")
		return
	}
	testUser.ID, err = primitive.ObjectIDFromHex(id)
	if err != nil {
		t.Errorf("error converting id")
		return
	}

	tempToken1, err := s.globalDBService.AddTempToken(models.TempToken{
		Expiration: time.Now().Unix() + 20,
		Purpose:    "survey-login",
		UserID:     testUser.ID.Hex(),
		InstanceID: testInstanceID,
		Info: map[string]string{
			"studyKey": "currentstudykey",
		},
	})
	if err != nil {
		t.Error(err)
		return
	}
	tempToken2, err := s.globalDBService.AddTempToken(models.TempToken{
		Expiration: time.Now().Unix() - 20,
		Purpose:    "survey-login",
		UserID:     testUser.ID.Hex(),
		InstanceID: testInstanceID,
		Info: map[string]string{
			"studyKey": "currentstudykey",
		},
	})
	if err != nil {
		t.Error(err)
		return
	}
	tempToken3, err := s.globalDBService.AddTempToken(models.TempToken{
		Expiration: time.Now().Unix() + 20,
		Purpose:    "unsubscribe-newsletter",
		UserID:     testUser.ID.Hex(),
		InstanceID: testInstanceID,
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("without payload", func(t *testing.T) {
		_, err := s.LoginWithTempToken(context.Background(), nil)
		ok, msg := shouldHaveGrpcErrorStatus(err, "invalid token")
		if !ok {
			t.Error(msg)
		}
	})

	t.Run("with empty payload", func(t *testing.T) {
		req := &api.JWTRequest{}
		_, err := s.LoginWithTempToken(context.Background(), req)
		ok, msg := shouldHaveGrpcErrorStatus(err, "invalid token")
		if !ok {
			t.Error(msg)
		}
	})

	t.Run("with wrong purpose token", func(t *testing.T) {
		req := &api.JWTRequest{
			Token: tempToken3,
		}
		_, err := s.LoginWithTempToken(context.Background(), req)
		ok, msg := shouldHaveGrpcErrorStatus(err, "invalid token")
		if !ok {
			t.Error(msg)
		}
	})

	t.Run("with expired token", func(t *testing.T) {
		req := &api.JWTRequest{
			Token: tempToken2,
		}
		_, err := s.LoginWithTempToken(context.Background(), req)
		ok, msg := shouldHaveGrpcErrorStatus(err, "invalid token")
		if !ok {
			t.Error(msg)
		}
	})

	t.Run("with valid token", func(t *testing.T) {
		req := &api.JWTRequest{
			Token: tempToken1,
		}
		resp, err := s.LoginWithTempToken(context.Background(), req)
		if err != nil {
			t.Errorf("unexpected error: %s", err.Error())
			return
		}
		if resp.RefreshToken != "" {
			t.Error("unexpected refreshtoken")
		}
		if resp.AccessToken == "" {
			t.Error("unexpected access token")
		}
		if len(resp.Profiles) != 1 && resp.Profiles[0].Alias != "test" {
			t.Error("unexpected profile")
		}
	})
}

func TestSignupWithEmail(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockMessagingClient := messageMock.NewMockMessagingServiceApiClient(mockCtrl)

	s := userManagementServer{
		userDBservice:   testUserDBService,
		globalDBService: testGlobalDBService,
		JWT: models.JWTConfig{
			TokenMinimumAgeMin:  time.Second * 1,
			TokenExpiryInterval: time.Second * 2,
		},
		clients: &models.APIClients{
			MessagingService: mockMessagingClient,
		},
	}

	wrongEmailFormatNewUserReq := &api.SignupWithEmailMsg{
		Email:             "test-signup",
		Password:          "SuperSecurePassword123!§$",
		InstanceId:        testInstanceID,
		PreferredLanguage: "en",
	}

	wrongPasswordFormatNewUserReq := &api.SignupWithEmailMsg{
		Email:             "test-signup@test.com",
		Password:          "short",
		InstanceId:        testInstanceID,
		PreferredLanguage: "en",
	}
	validNewUserReq := &api.SignupWithEmailMsg{
		Email:             "test-signup@test.com",
		Password:          "SuperSecurePassword123!§$",
		InstanceId:        testInstanceID,
		PreferredLanguage: "en",
	}

	t.Run("without payload", func(t *testing.T) {
		_, err := s.SignupWithEmail(context.Background(), nil)
		ok, msg := shouldHaveGrpcErrorStatus(err, "missing argument")
		if !ok {
			t.Error(msg)
		}
	})

	t.Run("with empty payload", func(t *testing.T) {
		req := &api.SignupWithEmailMsg{}
		_, err := s.SignupWithEmail(context.Background(), req)
		ok, msg := shouldHaveGrpcErrorStatus(err, "email not valid")
		if !ok {
			t.Error(msg)
		}
	})

	t.Run("with wrong email format", func(t *testing.T) {
		_, err := s.SignupWithEmail(context.Background(), wrongEmailFormatNewUserReq)
		ok, msg := shouldHaveGrpcErrorStatus(err, "email not valid")
		if !ok {
			t.Error(msg)
		}
	})

	t.Run("with wrong password format", func(t *testing.T) {
		_, err := s.SignupWithEmail(context.Background(), wrongPasswordFormatNewUserReq)
		ok, msg := shouldHaveGrpcErrorStatus(err, "password too weak")
		if !ok {
			t.Error(msg)
		}
	})

	t.Run("with valid fields", func(t *testing.T) {
		mockMessagingClient.EXPECT().SendInstantEmail(
			gomock.Any(),
			gomock.Any(),
		).Return(nil, nil)

		resp, err := s.SignupWithEmail(context.Background(), validNewUserReq)
		if err != nil {
			t.Errorf("unexpected error: %s", err.Error())
			return
		}
		if len(resp.AccessToken) < 1 || len(resp.RefreshToken) < 1 {
			t.Errorf("unexpected response: %s", resp)
			return
		}
		if len(resp.SelectedProfileId) < 1 {
			t.Errorf("unexpected selected profile: %s", resp.SelectedProfileId)
			return
		}
		if len(resp.Profiles) != 1 {
			t.Errorf("unexpected number of profiles: %d", len(resp.Profiles))
			return
		}
	})

	t.Run("with duplicate user (same email)", func(t *testing.T) {
		mockMessagingClient.EXPECT().SendInstantEmail(
			gomock.Any(),
			gomock.Any(),
		).Return(nil, nil)

		req := &api.SignupWithEmailMsg{
			Email:      "test-signup-1@test.com",
			Password:   "SuperSecurePassword123!§$",
			InstanceId: testInstanceID,
		}
		_, err := s.SignupWithEmail(context.Background(), req)
		if err != nil {
			t.Errorf("unexpected error: %s", err.Error())
			return
		}

		// Try to signup again:
		resp, err := s.SignupWithEmail(context.Background(), req)
		if err == nil || resp != nil {
			t.Errorf("should fail, when user exists already, wrong response: %s", resp)
			return
		}
	})
}

func TestSwitchProfileEndpoint(t *testing.T) {
	s := userManagementServer{
		userDBservice:   testUserDBService,
		globalDBService: testGlobalDBService,
		JWT: models.JWTConfig{
			TokenMinimumAgeMin:  time.Second * 1,
			TokenExpiryInterval: time.Second * 2,
		},
	}

	testUsers, err := addTestUsers([]models.User{
		{
			Account: models.Account{
				Type:          "email",
				AccountID:     "test_for_switch_profile@test.com",
				RefreshTokens: []string{"rt"},
			},
			Profiles: []models.Profile{
				{
					ID:    primitive.NewObjectID(),
					Alias: "main",
				},
				{
					ID:    primitive.NewObjectID(),
					Alias: "new1",
				},
				{
					ID:    primitive.NewObjectID(),
					Alias: "new2",
				},
			},
		},
	})
	if err != nil {
		t.Errorf("failed to create testusers: %s", err.Error())
		return
	}

	t.Run("without payload", func(t *testing.T) {
		_, err := s.SwitchProfile(context.Background(), nil)
		ok, msg := shouldHaveGrpcErrorStatus(err, "missing argument")
		if !ok {
			t.Error(msg)
		}
	})

	t.Run("with empty payload", func(t *testing.T) {
		req := &api.SwitchProfileRequest{}
		_, err := s.SwitchProfile(context.Background(), req)
		ok, msg := shouldHaveGrpcErrorStatus(err, "missing argument")
		if !ok {
			t.Error(msg)
		}
	})

	token := api.TokenInfos{
		Id:         testUsers[0].ID.Hex(),
		InstanceId: testInstanceID,
	}

	t.Run("with wrong profile id", func(t *testing.T) {
		req := &api.SwitchProfileRequest{
			Token:        &token,
			ProfileId:    "wrong_profile_id",
			RefreshToken: "rt",
		}
		_, err := s.SwitchProfile(context.Background(), req)
		ok, msg := shouldHaveGrpcErrorStatus(err, "profile not found")
		if !ok {
			t.Error(msg)
		}
	})

	t.Run("with correct profile id", func(t *testing.T) {
		req := &api.SwitchProfileRequest{
			Token:        &token,
			ProfileId:    testUsers[0].Profiles[2].ID.Hex(),
			RefreshToken: "rt",
		}
		resp, err := s.SwitchProfile(context.Background(), req)
		if err != nil {
			t.Errorf("unexpected error: %s", err.Error())
			return
		}
		if len(resp.AccessToken) < 1 || len(resp.RefreshToken) < 1 {
			t.Errorf("unexpected response: %s", resp)
			return
		}
		if resp.SelectedProfileId != testUsers[0].Profiles[2].ID.Hex() {
			t.Errorf("unexpected selected profile: %s", resp.SelectedProfileId)
			return
		}
		if len(resp.Profiles) != 3 {
			t.Errorf("unexpected number of profiles: %d", len(resp.Profiles))
			return
		}
	})
}

func TestVerifyAccountEndpoint(t *testing.T) {
	s := userManagementServer{
		userDBservice:   testUserDBService,
		globalDBService: testGlobalDBService,
		JWT: models.JWTConfig{
			TokenMinimumAgeMin:  time.Second * 1,
			TokenExpiryInterval: time.Second * 2,
		},
	}

	testUsers, err := addTestUsers([]models.User{
		{
			Account: models.Account{
				Type:      "email",
				AccountID: "test_for_verify_contact@test.com",
			},
			Profiles: []models.Profile{
				{
					ID:    primitive.NewObjectID(),
					Alias: "main",
				},
			},
			ContactInfos: []models.ContactInfo{
				{
					Type:  "email",
					Email: "test_for_verify_contact@test.com",
				},
				{
					Type:  "email",
					Email: "testadd@test.com",
				},
			},
		},
	})
	if err != nil {
		t.Errorf("failed to create testusers: %s", err.Error())
		return
	}

	t.Run("without payload", func(t *testing.T) {
		_, err := s.VerifyContact(context.Background(), nil)
		ok, msg := shouldHaveGrpcErrorStatus(err, "missing argument")
		if !ok {
			t.Error(msg)
		}
	})

	t.Run("with empty payload", func(t *testing.T) {
		req := &api.TempToken{}
		_, err := s.VerifyContact(context.Background(), req)
		ok, msg := shouldHaveGrpcErrorStatus(err, "missing argument")
		if !ok {
			t.Error(msg)
		}
	})

	t.Run("with wrong payload", func(t *testing.T) {
		req := &api.TempToken{
			Token: "wrong",
		}
		_, err := s.VerifyContact(context.Background(), req)
		ok, msg := shouldHaveGrpcErrorStatus(err, "wrong token")
		if !ok {
			t.Error(msg)
		}
	})

	t.Run("with wrong token purpose", func(t *testing.T) {
		tempTokenInfos := models.TempToken{
			UserID:     testUsers[0].ID.Hex(),
			InstanceID: testInstanceID,
			Purpose:    "wrong-purpose",
			Info: map[string]string{
				"type":  "email",
				"email": testUsers[0].Account.AccountID,
			},
			Expiration: tokens.GetExpirationTime(time.Hour * 24 * 30),
		}
		tempToken, err := s.globalDBService.AddTempToken(tempTokenInfos)
		if err != nil {
			t.Errorf("unexpected error: %s", err.Error())
			return
		}
		req := &api.TempToken{
			Token: tempToken,
		}

		_, err = s.VerifyContact(context.Background(), req)
		ok, msg := shouldHaveGrpcErrorStatus(err, "wrong token purpose")
		if !ok {
			t.Error(msg)
		}
	})

	t.Run("verify secondary address", func(t *testing.T) {
		tempTokenInfos := models.TempToken{
			UserID:     testUsers[0].ID.Hex(),
			InstanceID: testInstanceID,
			Purpose:    "contact-verification",
			Info: map[string]string{
				"type":  "email",
				"email": testUsers[0].ContactInfos[1].Email,
			},
			Expiration: tokens.GetExpirationTime(time.Hour * 24 * 30),
		}
		tempToken, err := s.globalDBService.AddTempToken(tempTokenInfos)
		if err != nil {
			t.Errorf("unexpected error: %s", err.Error())
			return
		}
		req := &api.TempToken{
			Token: tempToken,
		}

		resp, err := s.VerifyContact(context.Background(), req)
		if err != nil {
			t.Errorf("unexpected error: %s", err.Error())
			return
		}

		if resp.Account.AccountConfirmedAt > 0 {
			t.Error("account should not be confirmed")
		}
		if len(resp.ContactInfos) != 2 || resp.ContactInfos[1].ConfirmedAt < 1 {
			t.Error("email not confirmed yet")
		}
	})

	t.Run("verify main address", func(t *testing.T) {
		tempTokenInfos := models.TempToken{
			UserID:     testUsers[0].ID.Hex(),
			InstanceID: testInstanceID,
			Purpose:    "contact-verification",
			Info: map[string]string{
				"type":  "email",
				"email": testUsers[0].Account.AccountID,
			},
			Expiration: tokens.GetExpirationTime(time.Hour * 24 * 30),
		}
		tempToken, err := s.globalDBService.AddTempToken(tempTokenInfos)
		if err != nil {
			t.Errorf("unexpected error: %s", err.Error())
			return
		}
		req := &api.TempToken{
			Token: tempToken,
		}

		resp, err := s.VerifyContact(context.Background(), req)
		if err != nil {
			t.Errorf("unexpected error: %s", err.Error())
			return
		}

		if resp.Account.AccountConfirmedAt < 1 {
			t.Error("account not confirmed yet")
		}
		if len(resp.ContactInfos) != 2 || resp.ContactInfos[0].ConfirmedAt < 1 {
			t.Error("email not confirmed yet")
		}
	})
}