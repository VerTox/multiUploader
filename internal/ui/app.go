package ui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"

	"multiUploader/internal/config"
	"multiUploader/internal/localization"
	"multiUploader/internal/logging"
	"multiUploader/internal/providers"
	"multiUploader/internal/updater"
)

const (
	// GitHub repository для проверки обновлений
	githubOwner = "VerTox"
	githubRepo  = "multiUploader"
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
	// Создаем меню
	a.mainWindow.SetMainMenu(a.buildMenu())

	// Создаем вкладки
	a.uploadTab = NewUploadTab(a)
	a.settingsTab = NewSettingsTab(a)

	// Создаем контейнер с вкладками
	tabs := container.NewAppTabs(
		container.NewTabItem(localization.T("Upload"), a.uploadTab.Build()),
		container.NewTabItem(localization.T("Settings"), a.settingsTab.Build()),
	)

	// Устанавливаем содержимое окна
	a.mainWindow.SetContent(tabs)
}

// Run запускает приложение
func (a *App) Run() {
	// Применяем тему из конфигурации перед показом окна
	a.ApplyTheme()

	a.Build()

	// Проверяем обновления в фоне после запуска окна (не блокируем UI)
	go func() {
		// Ждем 2 секунды чтобы окно успело полностью отобразиться
		// (иначе диалог может появиться до готовности UI)
		time.Sleep(2 * time.Second)
		a.checkForUpdates(false) // false = не показывать сообщение если обновлений нет
	}()

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

// buildMenu создает главное меню приложения
func (a *App) buildMenu() *fyne.MainMenu {
	// File menu
	openLogsItem := fyne.NewMenuItem(localization.T("Open Logs Folder"), func() {
		a.openLogsFolder()
	})

	fileMenu := fyne.NewMenu(localization.T("File"),
		openLogsItem,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem(localization.T("Quit"), func() {
			a.fyneApp.Quit()
		}),
	)

	// Help menu
	checkUpdatesItem := fyne.NewMenuItem(localization.T("Check for Updates..."), func() {
		go a.checkForUpdates(true) // true = показывать сообщение даже если обновлений нет
	})

	aboutItem := fyne.NewMenuItem(localization.T("About"), func() {
		a.showAboutDialog()
	})

	helpMenu := fyne.NewMenu(localization.T("Help"), checkUpdatesItem, aboutItem)

	return fyne.NewMainMenu(fileMenu, helpMenu)
}

// openLogsFolder открывает папку с логами в файловом менеджере (кроссплатформенно)
func (a *App) openLogsFolder() {
	logDir := logging.GetLogDir()
	if logDir == "" {
		dialog.ShowInformation(localization.T("Logs Not Found"),
			localization.T("Could not determine logs location."),
			a.mainWindow)
		return
	}

	// Проверяем что директория существует
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		// Пытаемся создать
		if err := os.MkdirAll(logDir, 0755); err != nil {
			dialog.ShowInformation(localization.T("Error"),
				localization.T("Could not create logs directory:")+"\n"+err.Error(),
				a.mainWindow)
			return
		}
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", logDir)
	case "windows":
		cmd = exec.Command("explorer", logDir)
	default: // Linux и другие
		// Пробуем xdg-open (работает на большинстве Linux DE)
		cmd = exec.Command("xdg-open", logDir)
	}

	if err := cmd.Start(); err != nil {
		// Если не удалось открыть, показываем путь
		dialog.ShowInformation(localization.T("Logs Location"),
			localization.T("Could not open folder automatically.")+"\n\n"+localization.T("Logs are located at:")+"\n"+logDir,
			a.mainWindow)
	}
}

// SendNotification отправляет системное уведомление с учетом настроек и фокуса окна
func (a *App) SendNotification(title, content string) {
	// Получаем режим уведомлений из конфига
	globalCfg := a.config.GetGlobalConfig()
	mode := globalCfg.NotificationMode

	// Если уведомления выключены, ничего не делаем
	if mode == config.NotificationDisabled {
		return
	}

	// Если режим "только когда не в фокусе", проверяем фокус окна
	if mode == config.NotificationUnfocused {
		// Проверяем есть ли у canvas элемент в фокусе
		// Если canvas.Focused() != nil, значит окно активно и пользователь работает с ним
		if a.mainWindow.Canvas().Focused() != nil {
			// Окно в фокусе - не показываем уведомление
			return
		}
	}

	// Режим "always" или "unfocused" - отправляем уведомление
	a.fyneApp.SendNotification(&fyne.Notification{
		Title:   title,
		Content: content,
	})
}

// showAboutDialog показывает диалог "О программе" с информацией о версии
func (a *App) showAboutDialog() {
	// Получаем метаданные приложения из FyneApp.toml
	metadata := a.fyneApp.Metadata()

	message := fmt.Sprintf("%s v%s (Build %d)\n\n%s\n\n%s",
		metadata.Name,
		metadata.Version,
		metadata.Build,
		localization.T("A cross-platform file uploader for multiple hosting services."),
		localization.T("Copyright © 2026"),
	)

	dialog.ShowInformation(localization.T("About multiUploader"), message, a.mainWindow)
}

// checkForUpdates проверяет наличие новой версии на GitHub
// showNoUpdateMessage - если true, показывать сообщение даже если обновлений нет (для ручной проверки)
func (a *App) checkForUpdates(showNoUpdateMessage bool) {
	metadata := a.fyneApp.Metadata()
	currentVersion := metadata.Version

	// Проверяем обновления
	release, err := updater.CheckForUpdates(githubOwner, githubRepo, currentVersion)

	// Обновляем UI из горутины через fyne.Do
	if err != nil {
		if showNoUpdateMessage {
			dialog.ShowError(fmt.Errorf("failed to check for updates: %w", err), a.mainWindow)
		}
		return
	}

	if release != nil {
		// Есть новая версия - показываем диалог
		a.showUpdateDialog(release)
	} else if showNoUpdateMessage {
		// Обновлений нет, но пользователь запросил проверку вручную
		dialog.ShowInformation(localization.T("No Updates"),
			fmt.Sprintf(localization.T("You are using the latest version")+" (%s)", currentVersion),
			a.mainWindow)
	}
}

// showUpdateDialog показывает диалог о доступности новой версии
func (a *App) showUpdateDialog(release *updater.ReleaseInfo) {
	metadata := a.fyneApp.Metadata()

	message := fmt.Sprintf("%s\n\n%s v%s\n%s %s\n\n%s",
		localization.T("A new version is available!"),
		localization.T("Current version:"),
		metadata.Version,
		localization.T("New version:"),
		release.TagName,
		localization.T("Would you like to download it?"),
	)

	// Создаем custom dialog с кнопками
	dialog.ShowConfirm(localization.T("Update Available"), message, func(download bool) {
		if download {
			// Открываем страницу релиза в браузере
			a.openURL(release.HTMLURL)
		}
	}, a.mainWindow)
}

// openURL открывает URL в браузере (кроссплатформенно)
func (a *App) openURL(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default: // Linux
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Start(); err != nil {
		// Если не удалось открыть, показываем URL
		dialog.ShowInformation(localization.T("Download Link"),
			localization.T("Please visit:")+"\n"+url,
			a.mainWindow)
	}
}
