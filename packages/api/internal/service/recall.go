package service

import (
	"context"
	"encoding/json"

	apperrors "github.com/myindo/hlf-supply-chain/api/pkg/errors"
)

// IssueRecallInput is the typed payload for RecallService.Issue.
type IssueRecallInput struct {
	ID              string
	Scope           string
	TargetIDs       []string
	Reason          string
	Severity        string
	IssuedByName    string
	InstructionsURL string
}

// RecallService handles product-recall workflows.
type RecallService struct {
	gw FabricGateway
}

func NewRecallService(gw FabricGateway) *RecallService {
	return &RecallService{gw: gw}
}

// Issue files a recall covering one or more products/batches.
func (s *RecallService) Issue(_ context.Context, in IssueRecallInput) (*CreateResult, *apperrors.AppError) {
	targetJSON, err := json.Marshal(in.TargetIDs)
	if err != nil {
		return nil, apperrors.NewBadRequest("invalid targetIds")
	}
	if _, err := s.gw.SubmitTransaction("RecallContract:IssueRecall",
		in.ID, in.Scope, string(targetJSON), in.Reason,
		in.Severity, in.IssuedByName, in.InstructionsURL); err != nil {
		return nil, mapFabricError(err, "recall")
	}
	return &CreateResult{ID: in.ID}, nil
}

// Get returns a single recall record.
func (s *RecallService) Get(_ context.Context, id string) (json.RawMessage, *apperrors.AppError) {
	return evalRaw(s.gw, "recall", "RecallContract:ReadRecall", id)
}

// RecalledProducts lists all products currently under recall.
func (s *RecallService) RecalledProducts(_ context.Context) (json.RawMessage, *apperrors.AppError) {
	return evalRaw(s.gw, "recalled products", "RecallContract:GetRecalledProducts")
}
