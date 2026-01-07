package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"multiUploader/internal/config"
	"multiUploader/internal/providers"
)

// SettingsTab представляет вкладку настроек
type SettingsTab struct {
	app *App

	// Глобальные настройки
	themeSelect *widget.Select

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
	t.saveBtn = widget.NewButton("Save Settings", t.onSave)
	t.cancelBtn = widget.NewButton("Cancel", t.onCancel)

	// Кнопки в отдельном ряду
	buttonRow := container.NewHBox(
		layout.NewSpacer(),
		t.cancelBtn,
		t.saveBtn,
	)

	// Скроллируемый контент (БЕЗ кнопок)
	scrollContent := container.NewVBox(
		widget.NewLabel("Settings"),
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
	t.themeSelect = widget.NewSelect([]string{"auto", "light", "dark"}, nil)
	themeLabel := widget.NewLabel("Theme:")
	themeRow := container.NewBorder(nil, nil, themeLabel, nil, t.themeSelect)

	globalGroup := container.NewVBox(
		widget.NewLabelWithStyle("Global Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		themeRow,
	)

	return globalGroup
}

// buildProviderSettings создает секцию настроек провайдеров
func (t *SettingsTab) buildProviderSettings() fyne.CanvasObject {
	providerBoxes := container.NewVBox(
		widget.NewLabelWithStyle("Provider Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
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
			apiKeyLabel := widget.NewLabel("API Key:")
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
		enabledCheck: widget.NewCheck("Enabled", nil),
		apiKeyEntry:  widget.NewEntry(),
		statusLabel:  widget.NewLabel(""),
	}

	form.apiKeyEntry.SetPlaceHolder("Enter API key")

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
	t.themeSelect.SetSelected(globalCfg.Theme)

	// Загружаем настройки провайдеров
	for name, form := range t.providerForms {
		providerCfg := cfg.GetProviderConfig(name)

		form.enabledCheck.SetChecked(providerCfg.Enabled)
		form.apiKeyEntry.SetText(providerCfg.APIKey)
	}
}

// onSave обработчик сохранения настроек
func (t *SettingsTab) onSave() {
	cfg := t.app.Config()

	// Сохраняем глобальные настройки
	globalCfg := config.GlobalConfig{
		Theme: t.themeSelect.Selected,
	}
	cfg.SetGlobalConfig(globalCfg)

	// Сохраняем настройки провайдеров
	for name, form := range t.providerForms {
		providerCfg := config.ProviderConfig{
			Enabled: form.enabledCheck.Checked,
			APIKey:  form.apiKeyEntry.Text,
		}

		cfg.SetProviderConfig(name, providerCfg)
	}

	dialog.ShowInformation("Success", "Settings saved successfully!", t.app.MainWindow())

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
	dialog.ShowInformation("Cancelled", "Changes discarded", t.app.MainWindow())
}
