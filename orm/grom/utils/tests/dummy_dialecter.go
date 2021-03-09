package tests

import (
	"github.com/soedev/soelib/orm/grom"
	"github.com/soedev/soelib/orm/grom/clause"
	"github.com/soedev/soelib/orm/grom/logger"
	"github.com/soedev/soelib/orm/grom/schema"
)

type DummyDialector struct {
}

func (DummyDialector) Name() string {
	return "dummy"
}

func (DummyDialector) Initialize(*gorm.DB) error {
	return nil
}

func (DummyDialector) DefaultValueOf(field *schema.Field) clause.Expression {
	return clause.Expr{SQL: "DEFAULT"}
}

func (DummyDialector) Migrator(*gorm.DB) gorm.Migrator {
	return nil
}

func (DummyDialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {
	writer.WriteByte('?')
}

func (DummyDialector) QuoteTo(writer clause.Writer, str string) {
	writer.WriteByte('`')
	writer.WriteString(str)
	writer.WriteByte('`')
}

func (DummyDialector) Explain(sql string, vars ...interface{}) string {
	return logger.ExplainSQL(sql, nil, `"`, vars...)
}

func (DummyDialector) DataTypeOf(*schema.Field) string {
	return ""
}
