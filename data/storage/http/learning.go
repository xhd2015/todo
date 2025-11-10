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

// GetReadingPosition retrieves the saved reading position for a material
func (s *LearningMaterialsHttpService) GetReadingPosition(ctx context.Context, materialID int64) (int64, error) {
	// Create request payload
	req := struct {
		MaterialID int64 `json:"material_id"`
	}{
		MaterialID: materialID,
	}

	var response struct {
		MaterialID int64 `json:"material_id"`
		Offset     int64 `json:"offset"`
	}

	err := s.client.makeRequest(ctx, "/learning/recording/get", req, &response)
	if err != nil {
		return 0, fmt.Errorf("failed to get reading position: %w", err)
	}

	return response.Offset, nil
}

// UpdateReadingPosition saves the reading offset for a material
func (s *LearningMaterialsHttpService) UpdateReadingPosition(ctx context.Context, materialID int64, offset int64) error {
	// Create request payload
	req := struct {
		MaterialID int64 `json:"material_id"`
		Offset     int64 `json:"offset"`
	}{
		MaterialID: materialID,
		Offset:     offset,
	}

	var response struct {
		Success bool `json:"success"`
	}

	err := s.client.makeRequest(ctx, "/learning/recording/updateOffset", req, &response)
	if err != nil {
		return fmt.Errorf("failed to update reading offset: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("server reported failure to update reading offset")
	}

	return nil
}
