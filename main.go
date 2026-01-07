package main

import (
	"fyne.io/fyne/v2/app"

	"multiUploader/internal/providers"
	"multiUploader/internal/ui"
)

func main() {
	// Создаем Fyne приложение с уникальным ID для хранения настроек
	fyneApp := app.NewWithID("com.github.vertox.multiuploader")

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
