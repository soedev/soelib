package tenantdb

import (
	"errors"
	"github.com/jinzhu/gorm"
)

type TenantDataSource struct {
	Db              *gorm.DB `json:"-"`
	AutoID          int      `gorm:"primary_key" json:"autoId"`
	TenantID        int      `json:"tenantId"`
	TenantCode      string   `json:"tenantCode"`
	Version         int      `json:"version"`
	Name            string   `json:"name"`
	URL             string   `json:"url"`
	UserName        string   `json:"-"` //敏感数据，不传到前端
	Password        string   `json:"-"` //敏感数据，不传到前端
	DriverClassname string   `json:"driverClassname"`
	PoolSize        string   `json:"poolSize"`
	ExpMinute       string   `json:"expMinute"`
}

//TableName 设置表名
func (TenantDataSource) TableName() string {
	return "crm.tenant_datasource"
}

//GetByTenantID 根据租户号取得第一条数据源
func (t *TenantDataSource) GetByTenantID(tenantID string) (*TenantDataSource, error) {
	var tds []TenantDataSource
	t.Db.Where("tenant_id = ?", tenantID).Find(&tds)
	if len(tds) == 0 {
		return nil, errors.New("数据源未配置！")
	}
	if len(tds) > 1 {
		return nil, errors.New("数据源中找到多个配置，请检查！")
	}
	return &tds[0], nil
}
