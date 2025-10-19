package models

import "time"

type State struct {
	ID                  int64     `json:"id"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	ParentStateRecordID int64     `json:"parent_state_record_id"`
	Score               float64   `json:"score"`
	Scope               string    `json:"scope"`
	CreateTime          time.Time `json:"create_time"`
	UpdateTime          time.Time `json:"update_time"`
}

type StateOptional struct {
	ID                  *int64     `json:"id"`
	Name                *string    `json:"name"`
	Description         *string    `json:"description"`
	ParentStateRecordID *int64     `json:"parent_state_record_id"`
	Score               *float64   `json:"score"`
	Scope               *string    `json:"scope"`
	CreateTime          *time.Time `json:"create_time"`
	UpdateTime          *time.Time `json:"update_time"`
}

func (s *State) Update(optional *StateOptional) {
	if optional == nil {
		return
	}
	if optional.ID != nil {
		s.ID = *optional.ID
	}
	if optional.Name != nil {
		s.Name = *optional.Name
	}
	if optional.Description != nil {
		s.Description = *optional.Description
	}
	if optional.ParentStateRecordID != nil {
		s.ParentStateRecordID = *optional.ParentStateRecordID
	}
	if optional.Score != nil {
		s.Score = *optional.Score
	}
	if optional.Scope != nil {
		s.Scope = *optional.Scope
	}
	if optional.CreateTime != nil {
		s.CreateTime = *optional.CreateTime
	}
	if optional.UpdateTime != nil {
		s.UpdateTime = *optional.UpdateTime
	}
	s.UpdateTime = time.Now()
}

type StateEvent struct {
	ID            int64     `json:"id"`
	StateRecordID int64     `json:"state_record_id"`
	RecordData    string    `json:"record_data"`
	DeltaScore    float64   `json:"delta_score"`
	Description   string    `json:"description"`
	Details       string    `json:"details"`
	Scope         string    `json:"scope"`
	CreateTime    time.Time `json:"create_time"`
	UpdateTime    time.Time `json:"update_time"`
}

type StateEventOptional struct {
	ID            *int64     `json:"id"`
	StateRecordID *int64     `json:"state_record_id"`
	RecordData    *string    `json:"record_data"`
	DeltaScore    *float64   `json:"delta_score"`
	Description   *string    `json:"description"`
	Details       *string    `json:"details"`
	Scope         *string    `json:"scope"`
	CreateTime    *time.Time `json:"create_time"`
	UpdateTime    *time.Time `json:"update_time"`
}

func (se *StateEvent) Update(optional *StateEventOptional) {
	if optional == nil {
		return
	}
	if optional.ID != nil {
		se.ID = *optional.ID
	}
	if optional.StateRecordID != nil {
		se.StateRecordID = *optional.StateRecordID
	}
	if optional.RecordData != nil {
		se.RecordData = *optional.RecordData
	}
	if optional.DeltaScore != nil {
		se.DeltaScore = *optional.DeltaScore
	}
	if optional.Description != nil {
		se.Description = *optional.Description
	}
	if optional.Details != nil {
		se.Details = *optional.Details
	}
	if optional.Scope != nil {
		se.Scope = *optional.Scope
	}
	if optional.CreateTime != nil {
		se.CreateTime = *optional.CreateTime
	}
	if optional.UpdateTime != nil {
		se.UpdateTime = *optional.UpdateTime
	}
	se.UpdateTime = time.Now()
}
