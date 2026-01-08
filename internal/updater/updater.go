package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// GitHub API timeout
	apiTimeout = 10 * time.Second
)

// ReleaseInfo содержит информацию о релизе с GitHub
type ReleaseInfo struct {
	TagName string `json:"tag_name"` // например "v1.0.2"
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"` // ссылка на страницу релиза
}

// CheckForUpdates проверяет наличие новой версии на GitHub
// owner - владелец репозитория (например "vertox")
// repo - название репозитория (например "multiUploader")
// currentVersion - текущая версия (например "1.0.1")
// Возвращает информацию о последнем релизе или nil если обновлений нет
func CheckForUpdates(owner, repo, currentVersion string) (*ReleaseInfo, error) {
	// Формируем URL для GitHub API
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	// Создаем HTTP клиент с таймаутом
	client := &http.Client{
		Timeout: apiTimeout,
	}

	// Делаем запрос
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус код
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	// Парсим JSON
	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	// Сравниваем версии
	// TagName приходит в формате "v1.0.2", убираем "v"
	latestVersion := strings.TrimPrefix(release.TagName, "v")

	if CompareVersions(currentVersion, latestVersion) < 0 {
		// Текущая версия старее - есть обновление
		return &release, nil
	}

	// Обновлений нет
	return nil, nil
}

// CompareVersions сравнивает две версии в формате semantic versioning (major.minor.patch)
// Возвращает:
//
//	 1 если newVersion > currentVersion (новая версия новее)
//	 0 если версии равны
//	-1 если newVersion < currentVersion (новая версия старее)
func CompareVersions(currentVersion, newVersion string) int {
	// Убираем префикс "v" если есть
	currentVersion = strings.TrimPrefix(currentVersion, "v")
	newVersion = strings.TrimPrefix(newVersion, "v")

	// Парсим версии
	current := parseVersion(currentVersion)
	new := parseVersion(newVersion)

	// Сравниваем major
	if new[0] > current[0] {
		return 1
	} else if new[0] < current[0] {
		return -1
	}

	// Major равны, сравниваем minor
	if new[1] > current[1] {
		return 1
	} else if new[1] < current[1] {
		return -1
	}

	// Minor равны, сравниваем patch
	if new[2] > current[2] {
		return 1
	} else if new[2] < current[2] {
		return -1
	}

	// Версии равны
	return 0
}

// parseVersion парсит версию формата "major.minor.patch" в массив [major, minor, patch]
// Если формат невалидный, возвращает [0, 0, 0]
func parseVersion(version string) [3]int {
	parts := strings.Split(version, ".")
	result := [3]int{0, 0, 0}

	for i := 0; i < len(parts) && i < 3; i++ {
		if num, err := strconv.Atoi(parts[i]); err == nil {
			result[i] = num
		}
	}

	return result
}
