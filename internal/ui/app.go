package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"

	"multiUploader/internal/config"
	"multiUploader/internal/providers"
)

// ProviderFactory функция-фабрика для создания провайдера с API ключом
type ProviderFactory func(apiKey string) providers.Provider

// App представляет главное приложение
type App struct {
	fyneApp           fyne.App
	mainWindow        fyne.Window
	config            *config.ConfigManager
	providerFactories map[string]ProviderFactory
	uploadTab         *UploadTab
	settingsTab       *SettingsTab
}

// NewApp создает новое приложение
func NewApp(fyneApp fyne.App) *App {
	app := &App{
		fyneApp:           fyneApp,
		config:            config.NewConfigManager(fyneApp.Preferences()),
		providerFactories: make(map[string]ProviderFactory),
	}

	app.mainWindow = fyneApp.NewWindow("multiUploader")
	app.mainWindow.Resize(fyne.NewSize(700, 500))

	return app
}

// RegisterProviderFactory регистрирует фабрику провайдера в приложении
func (a *App) RegisterProviderFactory(name string, factory ProviderFactory) {
	a.providerFactories[name] = factory
}

// GetProvider создает и возвращает провайдер с актуальным API ключом из конфига
func (a *App) GetProvider(name string) (providers.Provider, bool) {
	factory, ok := a.providerFactories[name]
	if !ok {
		return nil, false
	}

	// Получаем актуальный API ключ из конфига
	apiKey := a.config.GetProviderAPIKey(name)

	// Создаем провайдер с актуальным ключом
	return factory(apiKey), true
}

// GetEnabledProviders возвращает список включенных провайдеров с актуальными API ключами
func (a *App) GetEnabledProviders() []providers.Provider {
	enabled := make([]providers.Provider, 0)
	for name, factory := range a.providerFactories {
		if a.config.IsProviderEnabled(name) {
			apiKey := a.config.GetProviderAPIKey(name)
			provider := factory(apiKey)
			enabled = append(enabled, provider)
		}
	}
	return enabled
}

// Build создает UI приложения
func (a *App) Build() {
	// Создаем вкладки
	a.uploadTab = NewUploadTab(a)
	a.settingsTab = NewSettingsTab(a)

	// Создаем контейнер с вкладками
	tabs := container.NewAppTabs(
		container.NewTabItem("Upload", a.uploadTab.Build()),
		container.NewTabItem("Settings", a.settingsTab.Build()),
	)

	// Устанавливаем содержимое окна
	a.mainWindow.SetContent(tabs)
}

// Run запускает приложение
func (a *App) Run() {
	// Применяем тему из конфигурации перед показом окна
	a.ApplyTheme()

	a.Build()
	a.mainWindow.ShowAndRun()
}

// Config возвращает менеджер конфигурации
func (a *App) Config() *config.ConfigManager {
	return a.config
}

// MainWindow возвращает главное окно приложения
func (a *App) MainWindow() fyne.Window {
	return a.mainWindow
}

// ApplyTheme применяет тему из конфигурации
func (a *App) ApplyTheme() {
	cfg := a.config.GetGlobalConfig()

	switch cfg.Theme {
	case "dark":
		a.fyneApp.Settings().SetTheme(theme.DarkTheme())
	case "light":
		a.fyneApp.Settings().SetTheme(theme.LightTheme())
	default:
		// "auto" или пустая строка - используем системную тему по умолчанию
		a.fyneApp.Settings().SetTheme(theme.DefaultTheme())
	}
}
