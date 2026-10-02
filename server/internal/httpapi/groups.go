package httpapi

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/bikkysamuel/splitsDemo/server/internal/auth"
	"github.com/bikkysamuel/splitsDemo/server/internal/groups"
	"github.com/bikkysamuel/splitsDemo/server/internal/httpapi/apigen"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

// Page sizes (doc 07 list convention).
const (
	defaultPageSize = 50
	maxPageSize     = 200
)

// ListGroups returns one page of the User's Groups.
func (s *Server) ListGroups(ctx context.Context, req apigen.ListGroupsRequestObject) (apigen.ListGroupsResponseObject, error) {
	p, err := mustPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	var after *platform.ID
	if req.Params.Cursor != nil {
		id, ok := decodeCursor(*req.Params.Cursor)
		if !ok {
			return apigen.ListGroups400ApplicationProblemPlusJSONResponse{
				BadRequestApplicationProblemPlusJSONResponse: apigen.BadRequestApplicationProblemPlusJSONResponse(problemInvalidCursor.problem("")),
			}, nil
		}
		after = &id
	}
	limit := defaultPageSize
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}
	if limit < 1 || limit > maxPageSize {
		return apigen.ListGroups400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: apigen.BadRequestApplicationProblemPlusJSONResponse(problemInvalidRequest.problem("limit must be 1–200")),
		}, nil
	}
	page, err := s.deps.Groups.List(ctx, p.UserID, after, limit)
	if err != nil {
		return nil, err
	}
	resp := apigen.ListGroups200JSONResponse{Items: make([]apigen.GroupSummary, len(page.Items))}
	for i, g := range page.Items {
		resp.Items[i] = apiGroupSummary(g)
	}
	if page.Next != nil {
		c := encodeCursor(*page.Next)
		resp.NextCursor = &c
	}
	return resp, nil
}

// CreateGroup makes a Group with the User as first Admin (FR-G1).
func (s *Server) CreateGroup(ctx context.Context, req apigen.CreateGroupRequestObject) (apigen.CreateGroupResponseObject, error) {
	p, err := mustPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	g, err := s.deps.Groups.Create(ctx, p.UserID, req.Body.Name, req.Body.Currency, req.Body.DisplayName)
	var invalid *groups.ValidationError
	switch {
	case errors.As(err, &invalid):
		return apigen.CreateGroup400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: groupsValidationProblem(invalid)}, nil
	case errors.Is(err, groups.ErrGroupLimit):
		return apigen.CreateGroup409ApplicationProblemPlusJSONResponse{
			ConflictApplicationProblemPlusJSONResponse: apigen.ConflictApplicationProblemPlusJSONResponse(problemGroupLimitReached.problem("")),
		}, nil
	case err != nil:
		return nil, err
	}
	return apigen.CreateGroup201JSONResponse(apiGroup(g)), nil
}

// GetGroup returns a Group the User is in; any other answers 404.
func (s *Server) GetGroup(ctx context.Context, req apigen.GetGroupRequestObject) (apigen.GetGroupResponseObject, error) {
	p, err := mustPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	g, err := s.deps.Groups.Get(ctx, p.UserID, platform.ID(req.GroupId))
	switch {
	case errors.Is(err, groups.ErrNotFound):
		return apigen.GetGroup404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: apigen.NotFoundApplicationProblemPlusJSONResponse(problemNotFound.problem("")),
		}, nil
	case err != nil:
		return nil, err
	}
	return apigen.GetGroup200JSONResponse(apiGroup(g)), nil
}

// RenameGroup renames a Group (Admins only, FR-G3).
func (s *Server) RenameGroup(ctx context.Context, req apigen.RenameGroupRequestObject) (apigen.RenameGroupResponseObject, error) {
	p, err := mustPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	g, err := s.deps.Groups.Rename(ctx, p.UserID, platform.ID(req.GroupId), req.Body.Name, int(req.Body.Version))
	var invalid *groups.ValidationError
	switch {
	case errors.As(err, &invalid):
		return apigen.RenameGroup400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: groupsValidationProblem(invalid)}, nil
	case errors.Is(err, groups.ErrNotFound):
		return apigen.RenameGroup404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: apigen.NotFoundApplicationProblemPlusJSONResponse(problemNotFound.problem("")),
		}, nil
	case errors.Is(err, groups.ErrAdminRequired):
		return apigen.RenameGroup403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: apigen.ForbiddenApplicationProblemPlusJSONResponse(problemAdminRequired.problem("")),
		}, nil
	case errors.Is(err, groups.ErrVersionConflict):
		return apigen.RenameGroup409ApplicationProblemPlusJSONResponse{
			ConflictApplicationProblemPlusJSONResponse: apigen.ConflictApplicationProblemPlusJSONResponse(problemVersionConflict.problem("")),
		}, nil
	case err != nil:
		return nil, err
	}
	return apigen.RenameGroup200JSONResponse(apiGroup(g)), nil
}

// mustPrincipal returns the signed-in User of a protected route; the auth
// middleware guarantees one.
func mustPrincipal(ctx context.Context) (auth.Principal, error) {
	p, ok := principalFrom(ctx)
	if !ok {
		return auth.Principal{}, errors.New("protected route reached without a principal")
	}
	return p, nil
}

// encodeCursor and decodeCursor keep cursors opaque: base64url of the last
// item's ID (doc 07).
func encodeCursor(id platform.ID) string { return base64.RawURLEncoding.EncodeToString(id[:]) }

func decodeCursor(c string) (platform.ID, bool) {
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil || len(b) != len(platform.ID{}) {
		return platform.ID{}, false
	}
	return platform.ID(b), true
}

func apiGroupSummary(g groups.Summary) apigen.GroupSummary {
	return apigen.GroupSummary{
		Id: openapi_types.UUID(g.ID), Name: g.Name, Currency: g.Currency, State: apigen.GroupState(g.State),
	}
}

func apiGroup(g groups.Group) apigen.Group {
	members := make([]apigen.Member, len(g.Members))
	for i, m := range g.Members {
		members[i] = apiMember(m)
	}
	return apigen.Group{
		Id:         openapi_types.UUID(g.ID),
		Name:       g.Name,
		Currency:   g.Currency,
		State:      apigen.GroupState(g.State),
		Version:    int32(g.Version), //nolint:gosec // a version
		MyMemberId: openapi_types.UUID(g.MyMemberID),
		Members:    members,
	}
}

func groupsValidationProblem(v *groups.ValidationError) apigen.BadRequestApplicationProblemPlusJSONResponse {
	p := problemValidationFailed.problem("")
	errs := make([]apigen.FieldError, len(v.Fields))
	for i, f := range v.Fields {
		errs[i] = apigen.FieldError{Field: "/" + f.Field, Code: f.Code}
	}
	p.Errors = &errs
	return apigen.BadRequestApplicationProblemPlusJSONResponse(p)
}

// AddMember adds a person by email or as a name-only Placeholder
// (FR-M1, FR-M2).
func (s *Server) AddMember(ctx context.Context, req apigen.AddMemberRequestObject) (apigen.AddMemberResponseObject, error) {
	p, err := mustPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	m, err := s.deps.Groups.AddMember(ctx, p.UserID, platform.ID(req.GroupId), req.Body.DisplayName, req.Body.Email)
	if err == nil {
		return apigen.AddMember201JSONResponse(apiMember(m)), nil
	}
	prob, ok := groupsProblem(err)
	if !ok {
		return nil, err
	}
	switch prob.Status {
	case http.StatusBadRequest:
		return apigen.AddMember400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: apigen.BadRequestApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusNotFound:
		return apigen.AddMember404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: apigen.NotFoundApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusConflict:
		return apigen.AddMember409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: apigen.ConflictApplicationProblemPlusJSONResponse(prob)}, nil
	}
	return nil, err
}

// UpdateMember makes a Member an Admin (Admins only, FR-G3).
func (s *Server) UpdateMember(ctx context.Context, req apigen.UpdateMemberRequestObject) (apigen.UpdateMemberResponseObject, error) {
	p, err := mustPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	m, err := s.deps.Groups.MakeAdmin(ctx, p.UserID, platform.ID(req.GroupId), platform.ID(req.MemberId), int(req.Body.Version))
	if err == nil {
		return apigen.UpdateMember200JSONResponse(apiMember(m)), nil
	}
	prob, ok := groupsProblem(err)
	if !ok {
		return nil, err
	}
	switch prob.Status {
	case http.StatusBadRequest:
		return apigen.UpdateMember400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: apigen.BadRequestApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusForbidden:
		return apigen.UpdateMember403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: apigen.ForbiddenApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusNotFound:
		return apigen.UpdateMember404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: apigen.NotFoundApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusConflict:
		return apigen.UpdateMember409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: apigen.ConflictApplicationProblemPlusJSONResponse(prob)}, nil
	}
	return nil, err
}

// groupsProblem maps an expected groups error to its problem; ok is false
// for anything else, which becomes a 500.
func groupsProblem(err error) (apigen.Problem, bool) {
	var invalid *groups.ValidationError
	switch {
	case errors.As(err, &invalid):
		return apigen.Problem(groupsValidationProblem(invalid)), true
	case errors.Is(err, groups.ErrNotFound):
		return problemNotFound.problem(""), true
	case errors.Is(err, groups.ErrAdminRequired):
		return problemAdminRequired.problem(""), true
	case errors.Is(err, groups.ErrVersionConflict):
		return problemVersionConflict.problem(""), true
	case errors.Is(err, groups.ErrGroupLimit):
		return problemGroupLimitReached.problem(""), true
	case errors.Is(err, groups.ErrMemberLimit):
		return problemMemberLimitReached.problem(""), true
	case errors.Is(err, groups.ErrMemberNotEligible):
		return problemMemberNotEligible.problem(""), true
	case errors.Is(err, groups.ErrGroupClosed):
		return problemGroupClosed.problem(""), true
	}
	return apigen.Problem{}, false
}

func apiMember(m groups.Member) apigen.Member {
	return apigen.Member{
		Id:          openapi_types.UUID(m.ID),
		DisplayName: m.DisplayName,
		Role:        apigen.MemberRole(m.Role),
		Status:      apigen.MemberStatus(m.Status),
		Placeholder: m.Placeholder(),
		JoinSeq:     int32(m.JoinSeq), //nolint:gosec // ≤ 50 Members
		Version:     int32(m.Version), //nolint:gosec // a version
	}
}
