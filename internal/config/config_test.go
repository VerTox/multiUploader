package config

import (
	"testing"
)

// mockPreferences реализует fyne.Preferences для тестирования
type mockPreferences struct {
	data map[string]interface{}
}

func newMockPreferences() *mockPreferences {
	return &mockPreferences{
		data: make(map[string]interface{}),
	}
}

func (m *mockPreferences) Bool(key string) bool {
	if v, ok := m.data[key].(bool); ok {
		return v
	}
	return false
}

func (m *mockPreferences) BoolWithFallback(key string, fallback bool) bool {
	if v, ok := m.data[key].(bool); ok {
		return v
	}
	return fallback
}

func (m *mockPreferences) SetBool(key string, value bool) {
	m.data[key] = value
}

func (m *mockPreferences) Float(key string) float64 {
	if v, ok := m.data[key].(float64); ok {
		return v
	}
	return 0
}

func (m *mockPreferences) FloatWithFallback(key string, fallback float64) float64 {
	if v, ok := m.data[key].(float64); ok {
		return v
	}
	return fallback
}

func (m *mockPreferences) SetFloat(key string, value float64) {
	m.data[key] = value
}

func (m *mockPreferences) Int(key string) int {
	if v, ok := m.data[key].(int); ok {
		return v
	}
	return 0
}

func (m *mockPreferences) IntWithFallback(key string, fallback int) int {
	if v, ok := m.data[key].(int); ok {
		return v
	}
	return fallback
}

func (m *mockPreferences) SetInt(key string, value int) {
	m.data[key] = value
}

func (m *mockPreferences) String(key string) string {
	if v, ok := m.data[key].(string); ok {
		return v
	}
	return ""
}

func (m *mockPreferences) StringWithFallback(key, fallback string) string {
	if v, ok := m.data[key].(string); ok {
		return v
	}
	return fallback
}

func (m *mockPreferences) SetString(key string, value string) {
	m.data[key] = value
}

func (m *mockPreferences) RemoveValue(key string) {
	delete(m.data, key)
}

func (m *mockPreferences) AddChangeListener(listener func()) {
	// Mock implementation - не делаем ничего для тестов
}

func (m *mockPreferences) BoolList(key string) []bool {
	if v, ok := m.data[key].([]bool); ok {
		return v
	}
	return nil
}

func (m *mockPreferences) BoolListWithFallback(key string, fallback []bool) []bool {
	if v, ok := m.data[key].([]bool); ok {
		return v
	}
	return fallback
}

func (m *mockPreferences) SetBoolList(key string, value []bool) {
	m.data[key] = value
}

func (m *mockPreferences) FloatList(key string) []float64 {
	if v, ok := m.data[key].([]float64); ok {
		return v
	}
	return nil
}

func (m *mockPreferences) FloatListWithFallback(key string, fallback []float64) []float64 {
	if v, ok := m.data[key].([]float64); ok {
		return v
	}
	return fallback
}

func (m *mockPreferences) SetFloatList(key string, value []float64) {
	m.data[key] = value
}

func (m *mockPreferences) IntList(key string) []int {
	if v, ok := m.data[key].([]int); ok {
		return v
	}
	return nil
}

func (m *mockPreferences) IntListWithFallback(key string, fallback []int) []int {
	if v, ok := m.data[key].([]int); ok {
		return v
	}
	return fallback
}

func (m *mockPreferences) SetIntList(key string, value []int) {
	m.data[key] = value
}

func (m *mockPreferences) StringList(key string) []string {
	if v, ok := m.data[key].([]string); ok {
		return v
	}
	return nil
}

func (m *mockPreferences) StringListWithFallback(key string, fallback []string) []string {
	if v, ok := m.data[key].([]string); ok {
		return v
	}
	return fallback
}

func (m *mockPreferences) SetStringList(key string, value []string) {
	m.data[key] = value
}

func (m *mockPreferences) ChangeListeners() []func() {
	// Mock implementation - возвращаем пустой список
	return nil
}

// TestGlobalConfig проверяет работу с глобальными настройками
func TestGlobalConfig(t *testing.T) {
	t.Run("Default theme", func(t *testing.T) {
		prefs := newMockPreferences()
		cm := NewConfigManager(prefs)

		config := cm.GetGlobalConfig()
		if config.Theme != "auto" {
			t.Errorf("Default theme = %s, want 'auto'", config.Theme)
		}
	})

	t.Run("Set and get theme", func(t *testing.T) {
		prefs := newMockPreferences()
		cm := NewConfigManager(prefs)

		// Устанавливаем тему
		config := GlobalConfig{Theme: "dark"}
		cm.SetGlobalConfig(config)

		// Читаем обратно
		savedConfig := cm.GetGlobalConfig()
		if savedConfig.Theme != "dark" {
			t.Errorf("Saved theme = %s, want 'dark'", savedConfig.Theme)
		}
	})

	t.Run("Update theme", func(t *testing.T) {
		prefs := newMockPreferences()
		cm := NewConfigManager(prefs)

		// Устанавливаем light
		cm.SetGlobalConfig(GlobalConfig{Theme: "light"})

		// Обновляем на dark
		cm.SetGlobalConfig(GlobalConfig{Theme: "dark"})

		config := cm.GetGlobalConfig()
		if config.Theme != "dark" {
			t.Errorf("Updated theme = %s, want 'dark'", config.Theme)
		}
	})
}

// TestProviderConfig проверяет работу с настройками провайдеров
func TestProviderConfig(t *testing.T) {
	t.Run("Default config", func(t *testing.T) {
		prefs := newMockPreferences()
		cm := NewConfigManager(prefs)

		config := cm.GetProviderConfig("TestProvider")
		if config.Enabled {
			t.Error("Default Enabled should be false")
		}
		if config.APIKey != "" {
			t.Errorf("Default APIKey = %s, want empty string", config.APIKey)
		}
	})

	t.Run("Set and get provider config", func(t *testing.T) {
		prefs := newMockPreferences()
		cm := NewConfigManager(prefs)

		// Устанавливаем настройки
		config := ProviderConfig{
			Enabled: true,
			APIKey:  "test-api-key-123",
		}
		cm.SetProviderConfig("DataVaults", config)

		// Читаем обратно
		savedConfig := cm.GetProviderConfig("DataVaults")
		if !savedConfig.Enabled {
			t.Error("Saved Enabled should be true")
		}
		if savedConfig.APIKey != "test-api-key-123" {
			t.Errorf("Saved APIKey = %s, want 'test-api-key-123'", savedConfig.APIKey)
		}
	})

	t.Run("Multiple providers", func(t *testing.T) {
		prefs := newMockPreferences()
		cm := NewConfigManager(prefs)

		// Настраиваем несколько провайдеров
		cm.SetProviderConfig("Rootz", ProviderConfig{
			Enabled: true,
			APIKey:  "rootz-key",
		})

		cm.SetProviderConfig("AkiraBox", ProviderConfig{
			Enabled: false,
			APIKey:  "akira-key",
		})

		// Проверяем что настройки не перепутались
		rootzConfig := cm.GetProviderConfig("Rootz")
		if !rootzConfig.Enabled || rootzConfig.APIKey != "rootz-key" {
			t.Error("Rootz config is incorrect")
		}

		akiraConfig := cm.GetProviderConfig("AkiraBox")
		if akiraConfig.Enabled || akiraConfig.APIKey != "akira-key" {
			t.Error("AkiraBox config is incorrect")
		}
	})

	t.Run("Update provider config", func(t *testing.T) {
		prefs := newMockPreferences()
		cm := NewConfigManager(prefs)

		// Изначально отключен
		cm.SetProviderConfig("TestProvider", ProviderConfig{
			Enabled: false,
			APIKey:  "old-key",
		})

		// Обновляем
		cm.SetProviderConfig("TestProvider", ProviderConfig{
			Enabled: true,
			APIKey:  "new-key",
		})

		config := cm.GetProviderConfig("TestProvider")
		if !config.Enabled {
			t.Error("Provider should be enabled after update")
		}
		if config.APIKey != "new-key" {
			t.Errorf("APIKey = %s, want 'new-key'", config.APIKey)
		}
	})
}

// TestIsProviderEnabled проверяет метод IsProviderEnabled
func TestIsProviderEnabled(t *testing.T) {
	prefs := newMockPreferences()
	cm := NewConfigManager(prefs)

	// По умолчанию выключен
	if cm.IsProviderEnabled("NewProvider") {
		t.Error("New provider should be disabled by default")
	}

	// Включаем
	cm.SetProviderConfig("NewProvider", ProviderConfig{Enabled: true})

	if !cm.IsProviderEnabled("NewProvider") {
		t.Error("Provider should be enabled")
	}

	// Выключаем
	cm.SetProviderConfig("NewProvider", ProviderConfig{Enabled: false})

	if cm.IsProviderEnabled("NewProvider") {
		t.Error("Provider should be disabled")
	}
}

// TestGetProviderAPIKey проверяет метод GetProviderAPIKey
func TestGetProviderAPIKey(t *testing.T) {
	prefs := newMockPreferences()
	cm := NewConfigManager(prefs)

	// По умолчанию пустой
	if key := cm.GetProviderAPIKey("TestProvider"); key != "" {
		t.Errorf("Default API key = %s, want empty string", key)
	}

	// Устанавливаем ключ
	cm.SetProviderConfig("TestProvider", ProviderConfig{
		APIKey: "my-secret-key",
	})

	if key := cm.GetProviderAPIKey("TestProvider"); key != "my-secret-key" {
		t.Errorf("API key = %s, want 'my-secret-key'", key)
	}
}

// TestConfigPersistence проверяет что настройки сохраняются
func TestConfigPersistence(t *testing.T) {
	prefs := newMockPreferences()
	cm := NewConfigManager(prefs)

	// Устанавливаем разные настройки
	cm.SetGlobalConfig(GlobalConfig{Theme: "dark"})
	cm.SetProviderConfig("Provider1", ProviderConfig{
		Enabled: true,
		APIKey:  "key1",
	})
	cm.SetProviderConfig("Provider2", ProviderConfig{
		Enabled: false,
		APIKey:  "key2",
	})

	// Создаём новый ConfigManager с теми же preferences
	cm2 := NewConfigManager(prefs)

	// Проверяем что все настройки сохранились
	globalConfig := cm2.GetGlobalConfig()
	if globalConfig.Theme != "dark" {
		t.Error("Global config not persisted")
	}

	p1Config := cm2.GetProviderConfig("Provider1")
	if !p1Config.Enabled || p1Config.APIKey != "key1" {
		t.Error("Provider1 config not persisted")
	}

	p2Config := cm2.GetProviderConfig("Provider2")
	if p2Config.Enabled || p2Config.APIKey != "key2" {
		t.Error("Provider2 config not persisted")
	}
}
