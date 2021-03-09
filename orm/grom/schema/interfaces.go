package schema

import (
	"github.com/soedev/soelib/orm/grom/clause"
)

type GormDataTypeInterface interface {
	GormDataType() string
}

type CreateClausesInterface interface {
	CreateClauses(*Field) []clause.Interface
}

type QueryClausesInterface interface {
	QueryClauses(*Field) []clause.Interface
}

type UpdateClausesInterface interface {
	UpdateClauses(*Field) []clause.Interface
}

type DeleteClausesInterface interface {
	DeleteClauses(*Field) []clause.Interface
}
