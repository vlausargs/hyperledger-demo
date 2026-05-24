package service

import (
	"github.com/myindo/hlf-supply-chain/api/internal/fabric"
)

type CAService struct {
	client *fabric.CAClient
}

func NewCAService(client *fabric.CAClient) *CAService {
	return &CAService{client: client}
}

func (s *CAService) Client() *fabric.CAClient {
	return s.client
}
