package soejwt

import (
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"strings"
)

//SubjectInfo 登录信息
type SubjectInfo struct {
	UserUID           string `json:"userUid"`
	TenantID          string `json:"tenantId"`
	TenantCode        string `json:"tenantCode"`
	AliUserPID        string `json:"aliUserPid"`
	AliMerchantPID    string `json:"aliMerchantPid"`
	AliAuthToken      string `json:"aliAuthToken"`
	AliAuthCode       string `json:"aliAuthCode"`
	OpenID            string `json:"openId"`
	OpenID2           string `json:"openId2"`
	LoginType         string `json:"loginType"`
	HoldShopCode      string `json:"holdShopCode"`
	AppID             string `json:"appId"`
	AliMerchantShopID string `json:"aliMerchantShopId"`
}

// LoginShopInfo 登录分店信息
type LoginShopInfo struct {
	ID        string
	Name      string
	Latitude  string
	Longitude string
	Address   string
	CoverURL  string
	Code      string
}

// LoginContent 登录内容
type LoginContent struct {
	//登录雇员
	EmployeeContent LoginEmployee

	//登录店家信息
	ShopContent LoginShopInfo
}

//LoginEmployee 员工信息
type LoginEmployee struct {
	ID          string
	WorkerID    string
	Name        string
	Phone       string
	Sex         uint
	Avatar      string
	RankName    string
	Permissions PermissionBitMaps
}

//PermissionBitMaps 仅限
type PermissionBitMaps struct {
	BitsMap []int
}

// UserLoginInfo 用户信息
type UserLoginInfo struct {
	Content    LoginContent
	NickName   string
	Telphone   string
	Token      string
	HeadImURL  string
	UserID     string
	TenantID   string
	TenantTags []string
}

//UnmarshalSubjectInfo 字符转对象
func UnmarshalSubjectInfo(info string) (subjectInfo SubjectInfo, err error) {
	err = json.Unmarshal([]byte(info), &subjectInfo)
	return subjectInfo, err
}

//UnmarshalUserLoginInfo 字符转对象
func UnmarshalUserLoginInfo(info string) (loginInfo UserLoginInfo, err error) {
	err = json.Unmarshal([]byte(info), &loginInfo)
	return loginInfo, err
}

//ParseJWT 解释登录信息
func ParseJWT(tokenStr string) (info SubjectInfo, err error) {
	info = SubjectInfo{}
	tokenStr = strings.Replace(tokenStr, "Bearer ", "", 1)
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte{195, 12, 44, 161, 231, 43}, nil
	})

	if err != nil {
		return info, fmt.Errorf("登录信息校验出错！%s", err.Error())
	}
	if !token.Valid {
		return info, fmt.Errorf("令牌校验出错！%s", err.Error())
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if n, ok := claims["sub"].(string); ok {
			var subjectInfo SubjectInfo
			subjectInfo, err = UnmarshalSubjectInfo(n)
			if err != nil {
				return info, fmt.Errorf("序列化用户出错！%s", err.Error())
			}
			return subjectInfo, nil
		}
		return info, fmt.Errorf("解析登录用户出错！%s", err.Error())
	}
	return info, fmt.Errorf("获取用户信息出错！%s", err.Error())
}

//NewJWTToken 生成token
func NewJWTToken(userID, subject string, expiresAt int64) (tokenStr string, err error) {
	// Create the Claims
	claims := &jwt.StandardClaims{
		Issuer:    userID,
		Subject:   subject, //claims["sub"]
		ExpiresAt: expiresAt,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err = token.SignedString(jwtKey)
	return tokenStr, err
}
