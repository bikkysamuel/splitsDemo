package httpapi

import (
	"context"
	"errors"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/bikkysamuel/splitsDemo/server/internal/auth"
	"github.com/bikkysamuel/splitsDemo/server/internal/httpapi/apigen"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

// SignUp creates an unverified User and returns a Session (FR-A1–A3).
func (s *Server) SignUp(ctx context.Context, req apigen.SignUpRequestObject) (apigen.SignUpResponseObject, error) {
	session, err := s.deps.Auth.SignUp(ctx, req.Body.Email, req.Body.Password)
	var invalid *auth.ValidationError
	switch {
	case errors.As(err, &invalid):
		return apigen.SignUp400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: apigen.BadRequestApplicationProblemPlusJSONResponse(validationProblem(invalid)),
		}, nil
	case errors.Is(err, auth.ErrEmailTaken):
		return apigen.SignUp409ApplicationProblemPlusJSONResponse{
			EmailTakenApplicationProblemPlusJSONResponse: apigen.EmailTakenApplicationProblemPlusJSONResponse(problemEmailTaken.problem("")),
		}, nil
	case err != nil:
		return nil, err
	}
	s.noteUser(ctx, session.User.ID)
	return apigen.SignUp201JSONResponse(authSession(session)), nil
}

// VerifyEmail checks the verification code (FR-A2).
func (s *Server) VerifyEmail(ctx context.Context, req apigen.VerifyEmailRequestObject) (apigen.VerifyEmailResponseObject, error) {
	session, err := s.deps.Auth.VerifyEmail(ctx, req.Body.Email, req.Body.Code)
	var invalid *auth.ValidationError
	switch {
	case errors.As(err, &invalid):
		return apigen.VerifyEmail400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: apigen.BadRequestApplicationProblemPlusJSONResponse(validationProblem(invalid)),
		}, nil
	case errors.Is(err, auth.ErrInvalidCode):
		return apigen.VerifyEmail400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: apigen.BadRequestApplicationProblemPlusJSONResponse(problemInvalidCode.problem("")),
		}, nil
	case err != nil:
		return nil, err
	}
	s.noteUser(ctx, session.User.ID)
	return apigen.VerifyEmail200JSONResponse(authSession(session)), nil
}

// ResendVerificationCode sends a new code; always 202 (doc 08).
func (s *Server) ResendVerificationCode(ctx context.Context, req apigen.ResendVerificationCodeRequestObject) (apigen.ResendVerificationCodeResponseObject, error) {
	err := s.deps.Auth.ResendVerificationCode(ctx, req.Body.Email)
	var invalid *auth.ValidationError
	switch {
	case errors.As(err, &invalid):
		return apigen.ResendVerificationCode400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: apigen.BadRequestApplicationProblemPlusJSONResponse(validationProblem(invalid)),
		}, nil
	case err != nil:
		return nil, err
	}
	return apigen.ResendVerificationCode202Response{}, nil
}

// SignIn checks the email and password and returns a new Session (FR-A4).
func (s *Server) SignIn(ctx context.Context, req apigen.SignInRequestObject) (apigen.SignInResponseObject, error) {
	session, err := s.deps.Auth.SignIn(ctx, req.Body.Email, req.Body.Password)
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		return apigen.SignIn401ApplicationProblemPlusJSONResponse{
			InvalidCredentialsApplicationProblemPlusJSONResponse: apigen.InvalidCredentialsApplicationProblemPlusJSONResponse(problemInvalidCredentials.problem("")),
		}, nil
	case err != nil:
		return nil, err
	}
	s.noteUser(ctx, session.User.ID)
	return apigen.SignIn200JSONResponse(authSession(session)), nil
}

// GetMe returns the signed-in User (FR-U1).
func (s *Server) GetMe(ctx context.Context, _ apigen.GetMeRequestObject) (apigen.GetMeResponseObject, error) {
	p, ok := principalFrom(ctx)
	if !ok {
		return nil, errors.New("GET /v1/me reached without a principal")
	}
	user, err := s.deps.Auth.Me(ctx, p)
	if err != nil {
		return nil, err
	}
	return apigen.GetMe200JSONResponse(apiUser(user)), nil
}

// noteUser puts the User's ID on the request log line of an anonymous auth
// endpoint once the User is known.
func (s *Server) noteUser(ctx context.Context, id platform.ID) {
	if info, ok := ctx.Value(requestInfoKey{}).(*requestInfo); ok {
		info.userID = &id
	}
}

func apiUser(u auth.User) apigen.User {
	return apigen.User{Id: openapi_types.UUID(u.ID), Email: u.Email, EmailVerified: u.EmailVerified}
}

func authSession(s auth.Session) apigen.AuthSession {
	return apigen.AuthSession{
		User:             apiUser(s.User),
		AccessToken:      s.AccessToken,
		AccessExpiresAt:  s.AccessExpiresAt,
		RefreshToken:     s.RefreshToken,
		RefreshExpiresAt: s.RefreshExpiresAt,
	}
}

// validationProblem lists each invalid field as a JSON Pointer to the
// request field (doc 07).
func validationProblem(v *auth.ValidationError) apigen.Problem {
	p := problemValidationFailed.problem("")
	errs := make([]apigen.FieldError, len(v.Fields))
	for i, f := range v.Fields {
		errs[i] = apigen.FieldError{Field: "/" + f.Field, Code: f.Code}
	}
	p.Errors = &errs
	return p
}
