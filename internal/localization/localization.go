package localization

import (
	"embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
)

//go:embed translations/*.json
var translationsFS embed.FS

// currentLocale хранит текущую выбранную локаль
var currentLocale = ""

// Init инициализирует систему локализации
// locale может быть "en", "ru" или "auto" (для использования системной локали)
func Init(locale string) error {
	// Устанавливаем текущую локаль
	SetLocale(locale)

	// Хак для переопределения системной локали
	// Обсуждение: https://github.com/fyne-io/fyne/issues/5333
	var content []byte
	var err error

	switch locale {
	case "en":
		content, err = translationsFS.ReadFile("translations/en.json")
	case "ru":
		content, err = translationsFS.ReadFile("translations/ru.json")
	case "auto":
		// Используем автоопределение - загружаем все переводы
		if err := lang.AddTranslationsFS(translationsFS, "translations"); err != nil {
			return err
		}
		return nil
	default:
		// Если неизвестная локаль, используем автоопределение
		if err := lang.AddTranslationsFS(translationsFS, "translations"); err != nil {
			return err
		}
		return nil
	}

	if err != nil {
		return err
	}

	// Регистрируем выбранный перевод под именем системной локали
	// Это заставляет Fyne использовать выбранный язык вместо системного
	if content != nil {
		name := lang.SystemLocale().LanguageString()
		return lang.AddTranslations(fyne.NewStaticResource(name+".json", content))
	}

	return nil
}

// SetLocale устанавливает текущую локаль приложения
func SetLocale(locale string) {
	currentLocale = locale
}

// GetLocale возвращает текущую локаль
func GetLocale() string {
	if currentLocale == "" || currentLocale == "auto" {
		return string(lang.SystemLocale())
	}
	return currentLocale
}

// T переводит строку с учетом текущей локали
// Это основная функция для перевода в приложении
func T(text string) string {
	return lang.L(text)
}

// GetAvailableLanguages возвращает список доступных языков для UI
func GetAvailableLanguages() []string {
	return []string{"Auto", "English", "Русский"}
}

// LanguageNameToCode конвертирует название языка в код локали
func LanguageNameToCode(name string) string {
	switch name {
	case "English":
		return "en"
	case "Русский":
		return "ru"
	default:
		return "auto"
	}
}

// LanguageCodeToName конвертирует код локали в название для UI
func LanguageCodeToName(code string) string {
	switch code {
	case "en":
		return "English"
	case "ru":
		return "Русский"
	default:
		return "Auto"
	}
}

// GetFyneLocale возвращает Fyne-совместимую локаль
func GetFyneLocale() fyne.Locale {
	locale := GetLocale()
	if locale == "auto" || locale == "" {
		return lang.SystemLocale()
	}
	return fyne.Locale(locale)
}
