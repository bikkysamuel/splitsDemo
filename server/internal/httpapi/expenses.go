package httpapi

import (
	"context"
	"errors"
	"net/http"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/bikkysamuel/splitsDemo/server/internal/expenses"
	"github.com/bikkysamuel/splitsDemo/server/internal/groups"
	"github.com/bikkysamuel/splitsDemo/server/internal/httpapi/apigen"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

// PreviewExpense returns the Shares an Expense would get (FR-E4).
func (s *Server) PreviewExpense(ctx context.Context, req apigen.PreviewExpenseRequestObject) (apigen.PreviewExpenseResponseObject, error) {
	p, err := mustPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.deps.Expenses.Preview(ctx, p.UserID, platform.ID(req.GroupId), expenseInput(*req.Body))
	if err == nil {
		resp := apigen.PreviewExpense200JSONResponse{
			Amount: money(c.Amount, c.Currency), OriginalAmount: money(c.Original.Amount, c.Original.Currency),
			ExchangeRate: c.Original.ExchangeRate, Shares: shareLines(c.Shares, c.Currency),
		}
		return resp, nil
	}
	prob, ok := expensesProblem(err)
	if !ok {
		return nil, err
	}
	switch prob.Status {
	case http.StatusBadRequest:
		return apigen.PreviewExpense400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: apigen.BadRequestApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusNotFound:
		return apigen.PreviewExpense404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: apigen.NotFoundApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusConflict:
		return apigen.PreviewExpense409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: apigen.ConflictApplicationProblemPlusJSONResponse(prob)}, nil
	}
	return nil, err
}

// CreateExpense records an Expense (FR-E1).
func (s *Server) CreateExpense(ctx context.Context, req apigen.CreateExpenseRequestObject) (apigen.CreateExpenseResponseObject, error) {
	p, err := mustPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	e, err := s.deps.Expenses.Create(ctx, p.UserID, platform.ID(req.GroupId), expenseInput(*req.Body))
	if err == nil {
		return apigen.CreateExpense201JSONResponse(apiExpense(e)), nil
	}
	prob, ok := expensesProblem(err)
	if !ok {
		return nil, err
	}
	switch prob.Status {
	case http.StatusBadRequest:
		return apigen.CreateExpense400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: apigen.BadRequestApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusNotFound:
		return apigen.CreateExpense404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: apigen.NotFoundApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusConflict:
		return apigen.CreateExpense409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: apigen.ConflictApplicationProblemPlusJSONResponse(prob)}, nil
	}
	return nil, err
}

// ListExpenses returns one page of a Group's Expenses, newest first.
func (s *Server) ListExpenses(ctx context.Context, req apigen.ListExpensesRequestObject) (apigen.ListExpensesResponseObject, error) {
	p, err := mustPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	bad := func(k problemKind, detail string) (apigen.ListExpensesResponseObject, error) {
		return apigen.ListExpenses400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: apigen.BadRequestApplicationProblemPlusJSONResponse(k.problem(detail))}, nil
	}
	var after *expenses.Cursor
	if req.Params.Cursor != nil {
		day, id, ok := decodeDatedCursor(*req.Params.Cursor)
		if !ok {
			return bad(problemInvalidCursor, "")
		}
		after = &expenses.Cursor{SpentOn: day, ID: id}
	}
	limit := defaultPageSize
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}
	if limit < 1 || limit > maxPageSize {
		return bad(problemInvalidRequest, "limit must be 1–200")
	}
	page, err := s.deps.Expenses.List(ctx, p.UserID, platform.ID(req.GroupId), after, limit)
	if errors.Is(err, groups.ErrNotFound) {
		return apigen.ListExpenses404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: apigen.NotFoundApplicationProblemPlusJSONResponse(problemNotFound.problem(""))}, nil
	}
	if err != nil {
		return nil, err
	}
	resp := apigen.ListExpenses200JSONResponse{Items: make([]apigen.ExpenseSummary, len(page.Items))}
	for i, e := range page.Items {
		resp.Items[i] = apigen.ExpenseSummary{
			Id: openapi_types.UUID(e.ID), PayerMemberId: openapi_types.UUID(e.PayerID), Amount: money(e.Amount, e.Currency),
			Category: apigen.Category(e.Category), Note: e.Note, SpentOn: openapi_types.Date{Time: e.SpentOn},
			State: apigen.ExpenseState(e.State),
		}
	}
	if page.Next != nil {
		c := encodeDatedCursor(page.Next.SpentOn, page.Next.ID)
		resp.NextCursor = &c
	}
	return resp, nil
}

// GetExpense returns an Expense with its Shares; 404 unless the User is in
// its Group.
func (s *Server) GetExpense(ctx context.Context, req apigen.GetExpenseRequestObject) (apigen.GetExpenseResponseObject, error) {
	p, err := mustPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	e, err := s.deps.Expenses.Get(ctx, p.UserID, platform.ID(req.ExpenseId))
	if errors.Is(err, expenses.ErrNotFound) {
		return apigen.GetExpense404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: apigen.NotFoundApplicationProblemPlusJSONResponse(problemNotFound.problem(""))}, nil
	}
	if err != nil {
		return nil, err
	}
	return apigen.GetExpense200JSONResponse(apiExpense(e)), nil
}

// EditExpense saves the creator's edit as the Expense's next revision
// (FR-E6).
func (s *Server) EditExpense(ctx context.Context, req apigen.EditExpenseRequestObject) (apigen.EditExpenseResponseObject, error) {
	p, err := mustPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	b := req.Body
	in := expenseInput(apigen.ExpenseInput{
		PayerMemberId: b.PayerMemberId, Amount: b.Amount, ExchangeRate: b.ExchangeRate, Category: b.Category,
		Note: b.Note, SpentOn: b.SpentOn, Split: b.Split,
	})
	e, err := s.deps.Expenses.Edit(ctx, p.UserID, platform.ID(req.ExpenseId), int(b.Version), in)
	if err == nil {
		return apigen.EditExpense200JSONResponse(apiExpense(e)), nil
	}
	prob, ok := expensesProblem(err)
	if !ok {
		return nil, err
	}
	switch prob.Status {
	case http.StatusBadRequest:
		return apigen.EditExpense400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: apigen.BadRequestApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusForbidden:
		return apigen.EditExpense403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: apigen.ForbiddenApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusNotFound:
		return apigen.EditExpense404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: apigen.NotFoundApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusConflict:
		return apigen.EditExpense409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: apigen.ConflictApplicationProblemPlusJSONResponse(prob)}, nil
	}
	return nil, err
}

// WithdrawExpense withdraws an Expense (its creator only; FR-E6).
func (s *Server) WithdrawExpense(ctx context.Context, req apigen.WithdrawExpenseRequestObject) (apigen.WithdrawExpenseResponseObject, error) {
	p, err := mustPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	e, err := s.deps.Expenses.Withdraw(ctx, p.UserID, platform.ID(req.ExpenseId), int(req.Body.Version))
	if err == nil {
		return apigen.WithdrawExpense200JSONResponse(apiExpense(e)), nil
	}
	prob, ok := expensesProblem(err)
	if !ok {
		return nil, err
	}
	switch prob.Status {
	case http.StatusForbidden:
		return apigen.WithdrawExpense403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: apigen.ForbiddenApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusNotFound:
		return apigen.WithdrawExpense404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: apigen.NotFoundApplicationProblemPlusJSONResponse(prob)}, nil
	case http.StatusConflict:
		return apigen.WithdrawExpense409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: apigen.ConflictApplicationProblemPlusJSONResponse(prob)}, nil
	}
	return nil, err
}

// expensesProblem maps an expected Expense error to its problem.
func expensesProblem(err error) (apigen.Problem, bool) {
	var invalid *expenses.ValidationError
	switch {
	case errors.As(err, &invalid):
		p := problemValidationFailed.problem("")
		errs := make([]apigen.FieldError, len(invalid.Fields))
		for i, f := range invalid.Fields {
			errs[i] = apigen.FieldError{Field: "/" + f.Field, Code: f.Code}
		}
		p.Errors = &errs
		return p, true
	case errors.Is(err, expenses.ErrGroupClosed):
		return problemGroupClosed.problem(""), true
	case errors.Is(err, expenses.ErrNotFound):
		return problemNotFound.problem(""), true
	case errors.Is(err, expenses.ErrNotCreator):
		return problemNotCreator.problem(""), true
	case errors.Is(err, expenses.ErrInvalidState):
		return problemInvalidState.problem(""), true
	case errors.Is(err, expenses.ErrVersionConflict):
		return problemVersionConflict.problem(""), true
	}
	return groupsProblem(err)
}

func expenseInput(b apigen.ExpenseInput) expenses.Input {
	in := expenses.Input{
		PayerID: platform.ID(b.PayerMemberId), Amount: b.Amount.Minor, Currency: b.Amount.Currency, ExchangeRate: b.ExchangeRate,
		Category: string(b.Category), Note: b.Note, SpentOn: b.SpentOn.Time, Method: string(b.Split.Method),
	}
	for _, m := range b.Split.Members {
		in.Members = append(in.Members, expenses.SplitEntry{MemberID: platform.ID(m.MemberId), Input: m.Input})
	}
	return in
}

func apiExpense(e expenses.Expense) apigen.Expense {
	return apigen.Expense{
		Id: openapi_types.UUID(e.ID), GroupId: openapi_types.UUID(e.GroupID),
		PayerMemberId: openapi_types.UUID(e.PayerID), CreatedByMemberId: openapi_types.UUID(e.CreatedBy),
		Amount: money(e.Amount, e.Currency), OriginalAmount: money(e.Original.Amount, e.Original.Currency),
		ExchangeRate: e.Original.ExchangeRate, Category: apigen.Category(e.Category), Note: e.Note,
		SpentOn: openapi_types.Date{Time: e.SpentOn}, SplitMethod: apigen.SplitMethod(e.Method),
		State: apigen.ExpenseState(e.State), Revision: int32(e.Revision), //nolint:gosec // a revision
		Version:   int32(e.Version), //nolint:gosec // a version
		CreatedAt: e.CreatedAt, Shares: shareLines(e.Shares, e.Currency),
	}
}

func money(minor int64, currency string) apigen.Money {
	return apigen.Money{Minor: minor, Currency: currency}
}

func shareLines(shares []expenses.Share, currency string) []apigen.ShareLine {
	lines := make([]apigen.ShareLine, len(shares))
	for i, s := range shares {
		lines[i] = apigen.ShareLine{MemberId: openapi_types.UUID(s.MemberID), Share: money(s.Amount, currency), Input: s.Input}
	}
	return lines
}
