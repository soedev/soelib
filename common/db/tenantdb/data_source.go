package tenantdb

import (
	"errors"
	"gorm.io/gorm"
	"time"
)

type (
	TenantDataSource struct {
		AutoID          int           `gorm:"primary_key" json:"autoId"`
		TenantID        int           `json:"tenantId"`
		TenantCode      string        `json:"tenantCode"`
		Version         int           `json:"version"`
		Name            string        `json:"name"`
		URL             string        `json:"url"`
		UserName        string        `json:"-"` //敏感数据，不传到前端
		Password        string        `json:"-"` //敏感数据，不传到前端
		DriverClassname string        `json:"driverClassname"`
		PoolSize        int           `json:"poolSize"` //空闲
		MaxPoolSize     int           `json:"maxPoolSize"`
		ExpMinute       time.Duration `json:"expMinute"`
	}
	TDSRepository struct {
		db *gorm.DB
	}
)

func NewTDSRepository(db *gorm.DB) *TDSRepository {
	return &TDSRepository{db: db}
}

// TableName 设置表名
func (TenantDataSource) TableName() string {
	return "crm.tenant_datasource"
}

// GetByTenantID 根据租户号取得第一条数据源
func (r *TDSRepository) GetByTenantID(tenantID string) (TenantDataSource, error) {
	var tds []TenantDataSource
	sql := `SELECT * FROM crm.tenant_datasource 
		INNER JOIN crm.client on crm.tenant_datasource.tenant_id=crm.client.tenant_id
		INNER JOIN crm.client_shop on crm.client.uid = crm.client_shop.client_uid 
		where crm.tenant_datasource.tenant_id=? or crm.client_shop.code=? limit 1`
	r.db.Raw(sql, tenantID, tenantID).Find(&tds)
	if len(tds) == 0 {
		return TenantDataSource{}, errors.New("数据源未配置！")
	}
	if len(tds) > 1 {
		return TenantDataSource{}, errors.New("数据源中找到多个配置，请检查！")
	}
	return tds[0], nil
}
