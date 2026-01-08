package main

import (
	"fyne.io/fyne/v2/app"

	"multiUploader/internal/localization"
	"multiUploader/internal/logging"
	"multiUploader/internal/providers"
	"multiUploader/internal/ui"
)

func main() {
	// Инициализируем логгер (пишет только errors в файл)
	if err := logging.Init(); err != nil {
		// Если не удалось инициализировать логгер, просто продолжаем
		// (приложение может работать без логов)
	}
	defer logging.Close()

	// Создаем Fyne приложение с уникальным ID для хранения настроек
	fyneApp := app.NewWithID("com.github.vertox.multiuploader")

	// Загружаем сохраненный язык из настроек (по умолчанию "auto")
	savedLanguage := fyneApp.Preferences().StringWithFallback("language", "auto")

	// Инициализируем систему локализации
	if err := localization.Init(savedLanguage); err != nil {
		// Если не удалось загрузить переводы, продолжаем с дефолтными значениями
		logging.Error("Failed to initialize localization: %v", err)
	}

	// Создаем наше приложение
	multiApp := ui.NewApp(fyneApp)

	// Регистрируем фабрики провайдеров
	// API ключи будут браться из конфига автоматически при каждом использовании

	// Мок провайдеры для тестирования UI (не требуют API ключ)
	//multiApp.RegisterProviderFactory("Mock Very Fast (100 MB/s)", func(apiKey string) providers.Provider {
	//	return providers.NewMockProvider("Mock Very Fast (100 MB/s)", 100)
	//})
	//multiApp.RegisterProviderFactory("Mock Fast (10 MB/s)", func(apiKey string) providers.Provider {
	//	return providers.NewMockProvider("Mock Fast (10 MB/s)", 10)
	//})
	//multiApp.RegisterProviderFactory("Mock Medium (2 MB/s)", func(apiKey string) providers.Provider {
	//	return providers.NewMockProvider("Mock Medium (2 MB/s)", 2)
	//})
	//multiApp.RegisterProviderFactory("Mock Slow (1 MB/s)", func(apiKey string) providers.Provider {
	//	return providers.NewMockProvider("Mock Slow (1 MB/s)", 1)
	//})

	// Реальные провайдеры
	multiApp.RegisterProviderFactory("DataVaults", func(apiKey string) providers.Provider {
		return providers.NewDataVaultsProvider(apiKey)
	})
	multiApp.RegisterProviderFactory("Rootz", func(apiKey string) providers.Provider {
		return providers.NewRootzProvider(apiKey)
	})
	multiApp.RegisterProviderFactory("AkiraBox", func(apiKey string) providers.Provider {
		return providers.NewAkiraBoxProvider(apiKey)
	})
	multiApp.RegisterProviderFactory("FileKeeper", func(apiKey string) providers.Provider {
		return providers.NewFileKeeperProvider(apiKey)
	})

	// Запускаем приложение
	multiApp.Run()
}
