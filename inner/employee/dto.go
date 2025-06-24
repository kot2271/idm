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

type CreateRequest struct {
	Name       string `json:"name" validate:"required,min=2,max=155" example:"Ivan Ivanov"`
	Email      string `json:"email" validate:"required,email" example:"ivan.ivanov@company.com"`
	Position   string `json:"position" validate:"required,min=2,max=100" example:"Developer"`
	Department string `json:"department" validate:"required,min=2,max=100" example:"IT"`
	RoleId     int64  `json:"role_id" validate:"required" example:"1"`
}

func (req *CreateRequest) ToEntity() Entity {
	return Entity{
		Name:       req.Name,
		Email:      req.Email,
		Position:   req.Position,
		Department: req.Department,
		RoleId:     req.RoleId,
	}
}

// PageRequest структура для запроса пагинации
type PageRequest struct {
	PageNumber int `json:"pageNumber" validate:"min=1"`
	PageSize   int `json:"pageSize" validate:"min=1,max=100"`
}

// PageResponse структура для ответа с пагинацией
type PageResponse struct {
	Data       []Response `json:"data"`
	PageNumber int        `json:"pageNumber"`
	PageSize   int        `json:"pageSize"`
	TotalCount int64      `json:"totalCount"`
	TotalPages int        `json:"totalPages"`
}
