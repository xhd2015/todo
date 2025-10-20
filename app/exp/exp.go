package exp

import "github.com/xhd2015/todo/data/mem"

type LogIDGroupMapping struct {
	ID      int64
	LogID   int64
	GroupID int64
}

func (c *LogIDGroupMapping) GetID() int64 {
	return c.ID
}
func (c *LogIDGroupMapping) SetID(id int64) {
	c.ID = id
}

var LogGroupStore *mem.MemStore[*LogIDGroupMapping]

func init() {
	// experimental
	store, err := mem.NewMemStore[*LogIDGroupMapping]("log_group.json")
	if err != nil {
		panic(err)
	}
	LogGroupStore = store
}

func GetMapping() map[int64]int64 {
	data, _, err := LogGroupStore.List(mem.Options{})
	if err != nil {
		panic(err)
	}
	m := make(map[int64]int64)
	for _, d := range data {
		m[d.LogID] = d.GroupID
	}
	return m
}
