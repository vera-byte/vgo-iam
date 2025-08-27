package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// AppType 应用类型
type AppType string

const (
	AppTypeWeb     AppType = "web"     // Web应用
	AppTypeMobile  AppType = "mobile"  // 移动应用
	AppTypeDesktop AppType = "desktop" // 桌面应用
	AppTypeAPI     AppType = "api"     // API应用
)

// AppStatus 应用状态
type AppStatus string

const (
	AppStatusActive    AppStatus = "active"    // 活跃
	AppStatusInactive  AppStatus = "inactive"  // 非活跃
	AppStatusSuspended AppStatus = "suspended" // 已暂停
)

// StringArray 字符串数组类型，用于PostgreSQL数组字段
type StringArray []string

// Value 实现driver.Valuer接口
func (sa StringArray) Value() (driver.Value, error) {
	if len(sa) == 0 {
		return nil, nil
	}
	return fmt.Sprintf("{\"%s\"}", fmt.Sprintf("%s", sa[0])), nil
}

// Scan 实现sql.Scanner接口
func (sa *StringArray) Scan(value interface{}) error {
	if value == nil {
		*sa = StringArray{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, sa)
	case string:
		return json.Unmarshal([]byte(v), sa)
	default:
		return fmt.Errorf("cannot scan %T into StringArray", value)
	}
}

// Application 应用信息
type Application struct {
	ID             int64       `json:"id" db:"id"`
	UserID         int64       `json:"user_id" db:"user_id"`
	AppName        string      `json:"app_name" db:"app_name"`
	AppDescription string      `json:"app_description" db:"app_description"`
	AppType        AppType     `json:"app_type" db:"app_type"`
	AppIconURL     string      `json:"app_icon_url" db:"app_icon_url"`
	AppWebsite     string      `json:"app_website" db:"app_website"`
	Status         AppStatus   `json:"status" db:"status"`
	CallbackURLs   StringArray `json:"callback_urls" db:"callback_urls"`
	AllowedOrigins StringArray `json:"allowed_origins" db:"allowed_origins"`
	CreatedAt      time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at" db:"updated_at"`

	// 关联信息
	UserName string `json:"user_name,omitempty" db:"user_name"`
}

// NewApplication 创建新应用
func NewApplication(userID int64, appName, appDescription string, appType AppType) *Application {
	now := time.Now()
	return &Application{
		UserID:         userID,
		AppName:        appName,
		AppDescription: appDescription,
		AppType:        appType,
		Status:         AppStatusActive,
		CallbackURLs:   StringArray{},
		AllowedOrigins: StringArray{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// IsActive 是否为活跃状态
func (app *Application) IsActive() bool {
	return app.Status == AppStatusActive
}

// IsInactive 是否为非活跃状态
func (app *Application) IsInactive() bool {
	return app.Status == AppStatusInactive
}

// IsSuspended 是否已暂停
func (app *Application) IsSuspended() bool {
	return app.Status == AppStatusSuspended
}

// SetStatus 设置应用状态
func (app *Application) SetStatus(status AppStatus) {
	app.Status = status
	app.UpdatedAt = time.Now()
}

// UpdateInfo 更新应用信息
func (app *Application) UpdateInfo(appName, appDescription, appIconURL, appWebsite string, appType AppType, callbackURLs, allowedOrigins []string) {
	app.AppName = appName
	app.AppDescription = appDescription
	app.AppIconURL = appIconURL
	app.AppWebsite = appWebsite
	app.AppType = appType
	app.CallbackURLs = StringArray(callbackURLs)
	app.AllowedOrigins = StringArray(allowedOrigins)
	app.UpdatedAt = time.Now()
}

// AddCallbackURL 添加回调URL
func (app *Application) AddCallbackURL(url string) {
	app.CallbackURLs = append(app.CallbackURLs, url)
	app.UpdatedAt = time.Now()
}

// RemoveCallbackURL 移除回调URL
func (app *Application) RemoveCallbackURL(url string) {
	for i, u := range app.CallbackURLs {
		if u == url {
			app.CallbackURLs = append(app.CallbackURLs[:i], app.CallbackURLs[i+1:]...)
			break
		}
	}
	app.UpdatedAt = time.Now()
}

// AddAllowedOrigin 添加允许的域名
func (app *Application) AddAllowedOrigin(origin string) {
	app.AllowedOrigins = append(app.AllowedOrigins, origin)
	app.UpdatedAt = time.Now()
}

// RemoveAllowedOrigin 移除允许的域名
func (app *Application) RemoveAllowedOrigin(origin string) {
	for i, o := range app.AllowedOrigins {
		if o == origin {
			app.AllowedOrigins = append(app.AllowedOrigins[:i], app.AllowedOrigins[i+1:]...)
			break
		}
	}
	app.UpdatedAt = time.Now()
}