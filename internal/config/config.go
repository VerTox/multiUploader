package config

import (
	"fyne.io/fyne/v2"
)

const (
	// Ключи для глобальных настроек
	keyTheme            = "global.theme"
	keyNotificationMode = "global.notification_mode"

	// Префиксы для настроек провайдеров
	prefixEnabled = ".enabled"
	prefixAPIKey  = ".api_key"
)

// NotificationMode определяет режим показа уведомлений
type NotificationMode string

const (
	// NotificationDisabled - уведомления отключены
	NotificationDisabled NotificationMode = "disabled"
	// NotificationUnfocused - уведомления только когда окно не в фокусе
	NotificationUnfocused NotificationMode = "unfocused"
	// NotificationAlways - уведомления всегда
	NotificationAlways NotificationMode = "always"
)

// GlobalConfig содержит глобальные настройки приложения
type GlobalConfig struct {
	// Theme тема приложения: "light", "dark", "auto"
	Theme string

	// NotificationMode режим показа уведомлений
	NotificationMode NotificationMode
}

// ProviderConfig содержит настройки для конкретного провайдера
type ProviderConfig struct {
	// Enabled включен ли провайдер
	Enabled bool

	// APIKey API ключ для провайдера
	APIKey string
}

// ConfigManager управляет настройками приложения
type ConfigManager struct {
	prefs fyne.Preferences
}

// NewConfigManager создает новый менеджер конфигурации
func NewConfigManager(prefs fyne.Preferences) *ConfigManager {
	return &ConfigManager{
		prefs: prefs,
	}
}

// GetGlobalConfig возвращает глобальные настройки
func (c *ConfigManager) GetGlobalConfig() GlobalConfig {
	theme := c.prefs.StringWithFallback(keyTheme, "auto")
	notificationMode := c.prefs.StringWithFallback(keyNotificationMode, string(NotificationUnfocused))

	return GlobalConfig{
		Theme:            theme,
		NotificationMode: NotificationMode(notificationMode),
	}
}

// SetGlobalConfig сохраняет глобальные настройки
func (c *ConfigManager) SetGlobalConfig(cfg GlobalConfig) {
	c.prefs.SetString(keyTheme, cfg.Theme)
	c.prefs.SetString(keyNotificationMode, string(cfg.NotificationMode))
}

// GetProviderConfig возвращает настройки для конкретного провайдера
func (c *ConfigManager) GetProviderConfig(providerName string) ProviderConfig {
	enabled := c.prefs.BoolWithFallback(providerName+prefixEnabled, false)
	apiKey := c.prefs.StringWithFallback(providerName+prefixAPIKey, "")

	return ProviderConfig{
		Enabled: enabled,
		APIKey:  apiKey,
	}
}

// SetProviderConfig сохраняет настройки для конкретного провайдера
func (c *ConfigManager) SetProviderConfig(providerName string, cfg ProviderConfig) {
	c.prefs.SetBool(providerName+prefixEnabled, cfg.Enabled)
	c.prefs.SetString(providerName+prefixAPIKey, cfg.APIKey)
}

// IsProviderEnabled проверяет, включен ли провайдер
func (c *ConfigManager) IsProviderEnabled(providerName string) bool {
	return c.prefs.BoolWithFallback(providerName+prefixEnabled, false)
}

// GetProviderAPIKey возвращает API ключ провайдера
func (c *ConfigManager) GetProviderAPIKey(providerName string) string {
	return c.prefs.StringWithFallback(providerName+prefixAPIKey, "")
}
