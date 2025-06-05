package employee

import "time"

type Entity struct {
	Id         int64     `db:"id"`
	Name       string    `db:"name"`
	Email      string    `db:"email"`
	Position   string    `db:"position"`
	Department string    `db:"department"`
	RoleId     int64     `db:"role_id"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

func (e *Entity) toResponse() Response {
	return Response{
		Id:         e.Id,
		Name:       e.Name,
		Email:      e.Email,
		Position:   e.Position,
		Department: e.Department,
		RoleId:     e.RoleId,
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
	}
}

type Response struct {
	Id         int64     `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Position   string    `json:"position"`
	Department string    `json:"department"`
	RoleId     int64     `json:"role_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
