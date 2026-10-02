package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"net/http"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/bikkysamuel/splitsDemo/server/internal/groups"
	"github.com/bikkysamuel/splitsDemo/server/internal/httpapi/apigen"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
	"github.com/bikkysamuel/splitsDemo/server/internal/settlements"
)

// RecordSettlement saves a Settlement (FR-S1, FR-S2).
func (s *Server) RecordSettlement(ctx context.Context, req apigen.RecordSettlementRequestObject) (apigen.RecordSettlementResponseObject, error) {
	p, err := mustPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	b := req.Body
	in := settlements.Input{
		From: platform.ID(b.FromMemberId), To: platform.ID(b.ToMemberId), Amount: b.Amount.Minor,
		Currency: b.Amount.Currency, SettledOn: b.SettledOn.Time, Note: b.Note,
		Acknowledge: b.AcknowledgeWarnings != nil && *b.AcknowledgeWarnings,
	}
	st, err := s.deps.Settlements.Record(ctx, p.UserID, platform.ID(req.GroupId), in)
	if err == nil {
		return apigen.RecordSettlement201JSONResponse(apiSettlement(st)), nil
	}
	prob, ok := settlementsProblem(err)
	if !ok {
		return nil, err
	}
	switch prob.Status {
	case http.StatusBadRequest:
		return apigen.RecordSettlement400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: apigen.BadRequestApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusNotFound:
		return apigen.RecordSettlement404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: apigen.NotFoundApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusConflict:
		return apigen.RecordSettlement409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: apigen.ConflictApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusUnprocessableEntity:
		return apigen.RecordSettlement422ApplicationProblemPlusJSONResponse{UnprocessableApplicationProblemPlusJSONResponse: apigen.UnprocessableApplicationProblemPlusJSONResponse(prob)}, nil
	}
	return nil, err
}

// ListSettlements returns one page of a Group's Settlements.
func (s *Server) ListSettlements(ctx context.Context, req apigen.ListSettlementsRequestObject) (apigen.ListSettlementsResponseObject, error) {
	p, err := mustPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	bad := func(k problemKind, detail string) (apigen.ListSettlementsResponseObject, error) {
		return apigen.ListSettlements400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: apigen.BadRequestApplicationProblemPlusJSONResponse(k.problem(detail))}, nil
	}
	var after *settlements.Cursor
	if req.Params.Cursor != nil {
		day, id, ok := decodeDatedCursor(*req.Params.Cursor)
		if !ok {
			return bad(problemInvalidCursor, "")
		}
		after = &settlements.Cursor{SettledOn: day, ID: id}
	}
	limit := defaultPageSize
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}
	if limit < 1 || limit > maxPageSize {
		return bad(problemInvalidRequest, "limit must be 1–200")
	}
	page, err := s.deps.Settlements.List(ctx, p.UserID, platform.ID(req.GroupId), after, limit)
	if errors.Is(err, groups.ErrNotFound) {
		return apigen.ListSettlements404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: apigen.NotFoundApplicationProblemPlusJSONResponse(problemNotFound.problem(""))}, nil
	}
	if err != nil {
		return nil, err
	}
	resp := apigen.ListSettlements200JSONResponse{Items: make([]apigen.Settlement, len(page.Items))}
	for i, st := range page.Items {
		resp.Items[i] = apiSettlement(st)
	}
	if page.Next != nil {
		c := encodeDatedCursor(page.Next.SettledOn, page.Next.ID)
		resp.NextCursor = &c
	}
	return resp, nil
}

// GetSettlement returns a Settlement; 404 unless the User is in its Group.
func (s *Server) GetSettlement(ctx context.Context, req apigen.GetSettlementRequestObject) (apigen.GetSettlementResponseObject, error) {
	p, err := mustPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	st, err := s.deps.Settlements.Get(ctx, p.UserID, platform.ID(req.SettlementId))
	if errors.Is(err, settlements.ErrNotFound) {
		return apigen.GetSettlement404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: apigen.NotFoundApplicationProblemPlusJSONResponse(problemNotFound.problem(""))}, nil
	}
	if err != nil {
		return nil, err
	}
	return apigen.GetSettlement200JSONResponse(apiSettlement(st)), nil
}

// WithdrawSettlement withdraws a Settlement (its creator only).
func (s *Server) WithdrawSettlement(ctx context.Context, req apigen.WithdrawSettlementRequestObject) (apigen.WithdrawSettlementResponseObject, error) {
	p, err := mustPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	st, err := s.deps.Settlements.Withdraw(ctx, p.UserID, platform.ID(req.SettlementId), int(req.Body.Version))
	if err == nil {
		return apigen.WithdrawSettlement200JSONResponse(apiSettlement(st)), nil
	}
	prob, ok := settlementsProblem(err)
	if !ok {
		return nil, err
	}
	switch prob.Status {
	case http.StatusForbidden:
		return apigen.WithdrawSettlement403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: apigen.ForbiddenApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusNotFound:
		return apigen.WithdrawSettlement404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: apigen.NotFoundApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusConflict:
		return apigen.WithdrawSettlement409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: apigen.ConflictApplicationProblemPlusJSONResponse(prob)}, nil
	}
	return nil, err
}

func settlementsProblem(err error) (apigen.Problem, bool) {
	var invalid *settlements.ValidationError
	var confirm *settlements.ConfirmationRequired
	switch {
	case errors.As(err, &invalid):
		p := problemValidationFailed.problem("")
		errs := make([]apigen.FieldError, len(invalid.Fields))
		for i, f := range invalid.Fields {
			errs[i] = apigen.FieldError{Field: "/" + f.Field, Code: f.Code}
		}
		p.Errors = &errs
		return p, true
	case errors.As(err, &confirm):
		p := problemConfirmationRequired.problem("")
		ws := make([]apigen.Warning, len(confirm.Warnings))
		for i, w := range confirm.Warnings {
			ws[i] = apigen.Warning{Code: w}
		}
		p.Warnings = &ws
		return p, true
	case errors.Is(err, settlements.ErrNotFound):
		return problemNotFound.problem(""), true
	case errors.Is(err, settlements.ErrGroupClosed):
		return problemGroupClosed.problem(""), true
	case errors.Is(err, settlements.ErrNotCreator):
		return problemNotCreator.problem(""), true
	case errors.Is(err, settlements.ErrInvalidState):
		return problemInvalidState.problem(""), true
	case errors.Is(err, settlements.ErrVersionConflict):
		return problemVersionConflict.problem(""), true
	}
	return groupsProblem(err)
}

func apiSettlement(s settlements.Settlement) apigen.Settlement {
	return apigen.Settlement{
		Id: openapi_types.UUID(s.ID), GroupId: openapi_types.UUID(s.GroupID),
		FromMemberId: openapi_types.UUID(s.From), ToMemberId: openapi_types.UUID(s.To),
		Amount: money(s.Amount, s.Currency), SettledOn: openapi_types.Date{Time: s.SettledOn}, Note: s.Note,
		CreatedByMemberId: openapi_types.UUID(s.CreatedBy), State: apigen.SettlementState(s.State),
		Version: int32(s.Version), CreatedAt: s.CreatedAt, //nolint:gosec // a version
	}
}

// Dated cursors (Expenses, Settlements): base64url of the last item's day
// (days since the Unix epoch, 4 bytes) and ID (16 bytes).
func encodeDatedCursor(day time.Time, id platform.ID) string {
	var b [20]byte
	binary.BigEndian.PutUint32(b[:4], uint32(day.Unix()/86400)) //nolint:gosec // dates after 1970
	copy(b[4:], id[:])
	return base64.RawURLEncoding.EncodeToString(b[:])
}

func decodeDatedCursor(s string) (time.Time, platform.ID, bool) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil || len(b) != 20 {
		return time.Time{}, platform.ID{}, false
	}
	days := int64(binary.BigEndian.Uint32(b[:4]))
	return time.Unix(days*86400, 0).UTC(), platform.ID(b[4:]), true
}
