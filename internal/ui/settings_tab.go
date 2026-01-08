package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"multiUploader/internal/config"
	"multiUploader/internal/localization"
	"multiUploader/internal/providers"
)

// SettingsTab представляет вкладку настроек
type SettingsTab struct {
	app *App

	// Глобальные настройки
	themeSelect            *widget.Select
	languageSelect         *widget.Select
	notificationRadioGroup *widget.RadioGroup

	// Настройки провайдеров
	providerForms map[string]*ProviderSettingsForm

	// Кнопки
	saveBtn   *widget.Button
	cancelBtn *widget.Button
}

// ProviderSettingsForm представляет форму настроек для одного провайдера
type ProviderSettingsForm struct {
	enabledCheck *widget.Check
	apiKeyEntry  *widget.Entry
	statusLabel  *widget.Label
}

// NewSettingsTab создает новую вкладку настроек
func NewSettingsTab(app *App) *SettingsTab {
	tab := &SettingsTab{
		app:           app,
		providerForms: make(map[string]*ProviderSettingsForm),
	}

	return tab
}

// Build создает UI вкладки настроек
func (t *SettingsTab) Build() fyne.CanvasObject {
	// Глобальные настройки
	globalSection := t.buildGlobalSettings()

	// Настройки провайдеров
	providerSection := t.buildProviderSettings()

	// Кнопки
	t.saveBtn = widget.NewButton(localization.T("Save Settings"), t.onSave)
	t.cancelBtn = widget.NewButton(localization.T("Cancel"), t.onCancel)

	// Кнопки в отдельном ряду
	buttonRow := container.NewHBox(
		layout.NewSpacer(),
		t.cancelBtn,
		t.saveBtn,
	)

	// Скроллируемый контент (БЕЗ кнопок)
	scrollContent := container.NewVBox(
		widget.NewLabel(localization.T("Settings")),
		widget.NewSeparator(),
		globalSection,
		widget.NewSeparator(),
		providerSection,
	)

	// Загружаем текущие настройки
	t.loadSettings()

	// Используем Border: скролл в центре, кнопки прибиты к низу
	return container.NewBorder(
		nil,                            // top
		container.NewPadded(buttonRow), // bottom - кнопки прибиты к низу
		nil,                            // left
		nil,                            // right
		container.NewScroll(container.NewPadded(scrollContent)), // center - скроллится
	)
}

// buildGlobalSettings создает секцию глобальных настроек
func (t *SettingsTab) buildGlobalSettings() fyne.CanvasObject {
	// Theme select
	themeOptions := []string{
		localization.T("auto"),
		localization.T("light"),
		localization.T("dark"),
	}
	t.themeSelect = widget.NewSelect(themeOptions, nil)
	themeLabel := widget.NewLabel(localization.T("Theme:"))
	themeRow := container.NewBorder(nil, nil, themeLabel, nil, t.themeSelect)

	// Language select
	t.languageSelect = widget.NewSelect(localization.GetAvailableLanguages(), nil)
	languageLabel := widget.NewLabel(localization.T("Language:"))
	languageRow := container.NewBorder(nil, nil, languageLabel, nil, t.languageSelect)

	// Notification settings
	notificationOptions := []string{
		localization.T("Disabled"),
		localization.T("Only when unfocused"),
		localization.T("Always"),
	}
	t.notificationRadioGroup = widget.NewRadioGroup(notificationOptions, nil)
	notificationLabel := widget.NewLabel(localization.T("Notifications:"))
	notificationBox := container.NewVBox(
		notificationLabel,
		t.notificationRadioGroup,
	)

	globalGroup := container.NewVBox(
		widget.NewLabelWithStyle(localization.T("Global Settings"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		themeRow,
		languageRow,
		notificationBox,
	)

	return globalGroup
}

// buildProviderSettings создает секцию настроек провайдеров
func (t *SettingsTab) buildProviderSettings() fyne.CanvasObject {
	providerBoxes := container.NewVBox(
		widget.NewLabelWithStyle(localization.T("Provider Settings"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	// Создаем форму для каждого провайдера
	for _, provider := range t.getAllProviders() {
		form := t.createProviderForm(provider)
		t.providerForms[provider.Name()] = form

		providerBox := container.NewVBox(
			widget.NewLabelWithStyle(provider.Name(), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			form.enabledCheck,
		)

		if provider.RequiresAuth() {
			apiKeyLabel := widget.NewLabel(localization.T("API Key:"))
			apiKeyRow := container.NewBorder(nil, nil, apiKeyLabel, nil, form.apiKeyEntry)
			providerBox.Add(apiKeyRow)
		}

		providerBox.Add(form.statusLabel)

		providerBoxes.Add(providerBox)
		providerBoxes.Add(widget.NewSeparator())
	}

	return providerBoxes
}

// createProviderForm создает форму настроек для провайдера
func (t *SettingsTab) createProviderForm(provider providers.Provider) *ProviderSettingsForm {
	form := &ProviderSettingsForm{
		enabledCheck: widget.NewCheck(localization.T("Enabled"), nil),
		apiKeyEntry:  widget.NewEntry(),
		statusLabel:  widget.NewLabel(""),
	}

	form.apiKeyEntry.SetPlaceHolder(localization.T("Enter API key"))

	return form
}

// getAllProviders возвращает все зарегистрированные провайдеры с актуальными API ключами
func (t *SettingsTab) getAllProviders() []providers.Provider {
	allProviders := make([]providers.Provider, 0, len(t.app.providerFactories))
	for name, factory := range t.app.providerFactories {
		apiKey := t.app.config.GetProviderAPIKey(name)
		provider := factory(apiKey)
		allProviders = append(allProviders, provider)
	}
	return allProviders
}

// loadSettings загружает текущие настройки из конфига
func (t *SettingsTab) loadSettings() {
	cfg := t.app.Config()

	// Загружаем глобальные настройки
	globalCfg := cfg.GetGlobalConfig()
	// Переводим значение темы для UI
	t.themeSelect.SetSelected(localization.T(globalCfg.Theme))

	// Загружаем язык из preferences
	savedLanguage := t.app.fyneApp.Preferences().StringWithFallback("language", "auto")
	languageName := localization.LanguageCodeToName(savedLanguage)
	t.languageSelect.SetSelected(languageName)

	// Конвертируем NotificationMode в UI текст
	notificationText := t.notificationModeToText(globalCfg.NotificationMode)
	t.notificationRadioGroup.SetSelected(notificationText)

	// Загружаем настройки провайдеров
	for name, form := range t.providerForms {
		providerCfg := cfg.GetProviderConfig(name)

		form.enabledCheck.SetChecked(providerCfg.Enabled)
		form.apiKeyEntry.SetText(providerCfg.APIKey)
	}
}

// notificationModeToText конвертирует NotificationMode в UI текст
func (t *SettingsTab) notificationModeToText(mode config.NotificationMode) string {
	switch mode {
	case config.NotificationDisabled:
		return localization.T("Disabled")
	case config.NotificationUnfocused:
		return localization.T("Only when unfocused")
	case config.NotificationAlways:
		return localization.T("Always")
	default:
		return localization.T("Only when unfocused")
	}
}

// textToNotificationMode конвертирует UI текст в NotificationMode
func (t *SettingsTab) textToNotificationMode(text string) config.NotificationMode {
	// Сравниваем с переведенными текстами
	if text == localization.T("Disabled") {
		return config.NotificationDisabled
	}
	if text == localization.T("Only when unfocused") {
		return config.NotificationUnfocused
	}
	if text == localization.T("Always") {
		return config.NotificationAlways
	}
	return config.NotificationUnfocused
}

// translatedThemeToCode конвертирует переведенное название темы в код
func (t *SettingsTab) translatedThemeToCode(text string) string {
	if text == localization.T("auto") {
		return "auto"
	}
	if text == localization.T("light") {
		return "light"
	}
	if text == localization.T("dark") {
		return "dark"
	}
	return "auto"
}

// onSave обработчик сохранения настроек
func (t *SettingsTab) onSave() {
	cfg := t.app.Config()

	// Проверяем, изменился ли язык
	savedLanguage := t.app.fyneApp.Preferences().StringWithFallback("language", "auto")
	newLanguageCode := localization.LanguageNameToCode(t.languageSelect.Selected)
	languageChanged := savedLanguage != newLanguageCode

	// Конвертируем выбранную тему обратно в код
	themeCode := t.translatedThemeToCode(t.themeSelect.Selected)

	// Сохраняем глобальные настройки
	globalCfg := config.GlobalConfig{
		Theme:            themeCode,
		NotificationMode: t.textToNotificationMode(t.notificationRadioGroup.Selected),
	}
	cfg.SetGlobalConfig(globalCfg)

	// Сохраняем язык в preferences
	t.app.fyneApp.Preferences().SetString("language", newLanguageCode)

	// Сохраняем настройки провайдеров
	for name, form := range t.providerForms {
		providerCfg := config.ProviderConfig{
			Enabled: form.enabledCheck.Checked,
			APIKey:  form.apiKeyEntry.Text,
		}

		cfg.SetProviderConfig(name, providerCfg)
	}

	// Показываем соответствующее сообщение
	if languageChanged {
		dialog.ShowInformation(localization.T("Language changed"),
			localization.T("Please restart the application to apply language changes"),
			t.app.MainWindow())
	} else {
		dialog.ShowInformation(localization.T("Success"), localization.T("Settings saved successfully!"), t.app.MainWindow())
	}

	// Применяем тему
	t.app.ApplyTheme()

	// Обновляем список провайдеров в Upload Tab
	if t.app.uploadTab != nil {
		t.app.uploadTab.Refresh()
	}
}

// onCancel обработчик отмены изменений
func (t *SettingsTab) onCancel() {
	t.loadSettings()
	dialog.ShowInformation(localization.T("Cancelled"), localization.T("Changes discarded"), t.app.MainWindow())
}
