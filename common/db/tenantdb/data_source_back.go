package tenantdb

import (
	"errors"
	"github.com/jinzhu/gorm"
	"time"
)

type TenantDataSourceBack struct {
	Db              *gorm.DB      `gorm:"-" json:"-"`
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
	Enable          int           `json:"-"`
}

func (TenantDataSourceBack) TableName() string {
	return "crm.tenant_datasource_back"
}

//GetByTenantID 根据租户号取得第一条数据源
func (t *TenantDataSourceBack) GetByTenantID(tenantID string) (*TenantDataSourceBack, error) {
	var tds []TenantDataSourceBack
	sql := `SELECT * FROM crm.tenant_datasource_back 
		INNER JOIN crm.client on crm.tenant_datasource_back.tenant_id=crm.client.tenant_id
		INNER JOIN crm.client_shop on crm.client.uid = crm.client_shop.client_uid 
		where crm.tenant_datasource_back.tenant_id=? or crm.client_shop.code=? limit 1`
	t.Db.Raw(sql, tenantID,tenantID).Find(&tds)
	if len(tds) == 0 {
		return nil, nil
	}
	if len(tds) > 1 {
		return nil, errors.New("数据源中找到多个配置，请检查！")
	}
	return &tds[0], nil
}
