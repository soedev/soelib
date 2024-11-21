package tenantdb

import (
	"fmt"
	"github.com/soedev/soelib/common/db/specialdb"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"testing"
	"time"
)

func TestGetDbFromMapWithOpt(t *testing.T) {
	dbInfo := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable password=%s application_name=go-soelib",
		"192.168.1.208", 31209, "soe", "soeoadb", "soesoft")
	config := specialdb.DbConfig{
		DBInfo:          dbInfo,
		Dialect:         "postgres",
		DBConfig:        gorm.Config{Logger: logger.Default.LogMode(logger.Info)},
		MaxIdleConns:    1,
		MaxOpenConns:    50,
		ConnMaxLifetime: time.Minute * 5,
	}
	CrmDb, err := specialdb.ConnDB(config)
	if err != nil {
		t.Fatal(err)
	}
	db, err := GetDbFromMapWithOpt("600002", CrmDb, &OptSQL{
		DBConfig:        gorm.Config{Logger: logger.Default.LogMode(logger.Info)},
		ApplicationName: "soe-lib",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Exec("select 1").Error
	if err != nil {
		t.Fatal(err)
	}
}
