package http

import (
	"context"
	"fmt"

	"github.com/xhd2015/todo/models"
)

// LearningMaterialsHttpService implements learning materials service
type LearningMaterialsHttpService struct {
	client *Client
}

func NewLearningMaterialsService(client *Client) *LearningMaterialsHttpService {
	return &LearningMaterialsHttpService{client: client}
}

func (s *LearningMaterialsHttpService) ListMaterials(ctx context.Context, offset int, limit int) ([]*models.LearningMaterial, int64, error) {
	// Create request payload
	req := struct {
		Offset int `json:"offset"`
		Limit  int `json:"limit"`
	}{
		Offset: offset,
		Limit:  limit,
	}

	var response struct {
		Data  []*models.LearningMaterial `json:"data"`
		Count int64                      `json:"count"`
	}

	err := s.client.makeRequest(ctx, "/learning/list", req, &response)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list learning materials: %w", err)
	}

	return response.Data, response.Count, nil
}

type MaterialContentResponse struct {
	Content    string `json:"content"`
	TotalBytes int    `json:"total_bytes"`
	HasMore    bool   `json:"has_more"`
	LastOffset int64  `json:"last_offset"`
}

func (s *LearningMaterialsHttpService) GetMaterialContent(ctx context.Context, id int64, offset int, limit int) (*MaterialContentResponse, error) {
	// Create request payload
	req := struct {
		ID     int64 `json:"id"`
		Offset int   `json:"offset"`
		Limit  int   `json:"limit"`
	}{
		ID:     id,
		Offset: offset,
		Limit:  limit,
	}

	var response MaterialContentResponse

	err := s.client.makeRequest(ctx, "/learning/content", req, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get material content: %w", err)
	}

	return &response, nil
}
