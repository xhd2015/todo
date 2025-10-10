package models

import "time"

type Happening struct {
	ID         int64     `json:"id"`
	Content    string    `json:"content"`
	CreateTime time.Time `json:"create_time"`
	UpdateTime time.Time `json:"update_time"`
}

type HappeningOptional struct {
	ID         *int64     `json:"id"`
	Content    *string    `json:"content"`
	CreateTime *time.Time `json:"create_time"`
	UpdateTime *time.Time `json:"update_time"`
}

func (c *Happening) Update(optional *HappeningOptional) {
	if optional == nil {
		return
	}
	if optional.ID != nil {
		c.ID = *optional.ID
	}
	if optional.Content != nil {
		c.Content = *optional.Content
	}
	if optional.CreateTime != nil {
		c.CreateTime = *optional.CreateTime
	}
	if optional.UpdateTime != nil {
		c.UpdateTime = *optional.UpdateTime
	}
	c.UpdateTime = time.Now()
}
