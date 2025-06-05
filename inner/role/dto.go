package role

import "time"

type Entity struct {
	Id        int64     `db:"id"`
	Name      string    `db:"name"`
	Desc      string    `db:"description"`
	Status    bool      `db:"status"`
	ParentId  *int64    `db:"parent_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (e *Entity) toResponse() Response {
	return Response{
		Id:        e.Id,
		Name:      e.Name,
		Desc:      e.Desc,
		Status:    e.Status,
		ParentId:  e.ParentId,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

type Response struct {
	Id        int64     `json:"id"`
	Name      string    `json:"name"`
	Desc      string    `json:"description"`
	Status    bool      `json:"status"`
	ParentId  *int64    `json:"parent_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
