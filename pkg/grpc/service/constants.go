package service

const (
	contactVerificationMessageCooldown = 1 * 60 // Minimum delay between 2 verification code sending for a new contact, seconds
	loginVerificationCodeCooldown      = 20 // Minimum delay between 2 verification code sending for a new login, in seconds

	// Window time period to count event and limit rrate
	signupRateLimitWindow           = 5 * 60  // to count the new signup, seconds
	loginFailedAttemptWindow        = 5 * 50  // to count the login failure, seconds
	passwordResetAttemptWindow      = 60 * 60 // to count the password failure, in seconds, default=1 hour
	allowedPasswordAttempts         = 10
	allowedVerificationCodeAttempts = 3

	userCreationTimestampOffset = 7 * 24 * 3600 // consider user deletion only after this time, when created by admin

	maximumProfilesAllowed = 6
)
