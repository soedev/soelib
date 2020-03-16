package soejwt

import (
	"encoding/json"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"strings"
	"time"
)

/**
  统一的 soe soejwt 处理
*/

const jwtKey = "www.soe.crm"

var jwtSecret = []byte(jwtKey)

//Claims JWT声明
type Claims struct {
	Username string `json:"username"`
	Password string `json:"password"`
	jwt.StandardClaims
}

//SoeAuthToken 解析autoToken信息
type SoeAuthToken struct {
	UserUID     string `json:"userUid"`
	MobilePhone string `json:"mobilePhone"` //手机
	NickName    string `json:"nickName"`    //昵称
	WxUnionID   string `json:"wxUnionId"`   //唯一码
	OpenID      string `json:"openId"`
	OpenID2     string `json:"openId2"`
	AliID       string `json:"aliId"`
	HeadImgUrl  string `json:"headImgUrl"`
}

//GenerateToken 生成软件登录用的token
func GenerateToken(username, password, issuer string) (string, error) {
	nowTime := time.Now()
	expireTime := nowTime.Add(24 * time.Hour)
	claims := Claims{
		username,
		password,
		jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
			Issuer:    issuer,
		},
	}
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenClaims.SignedString(jwtSecret)
	return token, err
}

//ParseToken 解析token
func ParseToken(token string) (*Claims, error) {
	token = strings.ReplaceAll(token, "Bearer ", "")
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if tokenClaims != nil {
		if claims, ok := tokenClaims.Claims.(*Claims); ok && tokenClaims.Valid {
			return claims, nil
		}
	}
	return nil, err
}
func secret() jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		return []byte(viper.GetString(jwtKey)), nil
	}
}

//GetSoeAuthToken 获取AuthToken信息
func GetSoeAuthToken(authToken string) (soeAuthToken SoeAuthToken, err error) {
	authToken = strings.ReplaceAll(authToken, "Bearer ", "")
	token, _ := jwt.Parse(authToken, secret())
	claims, _ := token.Claims.(jwt.MapClaims)

	if claims == nil {
		return soeAuthToken, errors.New("无效的authToken")
	}
	if claims["sub"].(string) == "" {
		return soeAuthToken, errors.New("authToken格式错误")
	} else {
		subJson := claims["sub"].(string)
		err := json.Unmarshal([]byte(subJson), &soeAuthToken)
		if err != nil {
			return soeAuthToken, err
		}
	}
	return soeAuthToken, nil
}

//GenerateSoeAuthToken 生成TOKEN
func GenerateSoeAuthToken(subjectInfo SubjectInfo) (string, error) {
	isSuer := ""
	if subjectInfo.AppID == "" {
		isSuer = subjectInfo.TenantID
	}
	uuid := uuid.New().String()
	authSession := strings.ReplaceAll(uuid, "-", "")
	subjectInfoJson, _ := json.Marshal(subjectInfo)
	claims := make(jwt.MapClaims)
	claims["iss"] = isSuer
	claims["jti"] = authSession
	claims["sub"] = string(subjectInfoJson)
	tokenClaims := jwt.New(jwt.SigningMethodHS256)
	tokenClaims.Claims = claims
	token, err := tokenClaims.SignedString([]byte(viper.GetString(jwtKey)))
	if err != nil {
		return "", err
	}
	return token, nil
}
