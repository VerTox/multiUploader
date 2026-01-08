package ui

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"multiUploader/internal/localization"
	"multiUploader/internal/logging"
	"multiUploader/internal/providers"
)

// UploadTab представляет вкладку загрузки файлов
type UploadTab struct {
	app *App

	// UI элементы
	providerSelect *widget.Select
	filePathLabel  *widget.Label
	selectFileBtn  *widget.Button
	uploadBtn      *widget.Button
	progressBar    *widget.ProgressBar
	speedLabel     *widget.Label
	uploadedLabel  *widget.Label
	etaLabel       *widget.Label
	resultLabel    *widget.Label

	// Data bindings (потокобезопасные)
	progressBinding  binding.Float
	uploadedBinding  binding.String
	speedBinding     binding.String
	etaBinding       binding.String
	resultBinding    binding.String
	uploadBtnBinding binding.String

	// Состояние
	selectedFile     fyne.URI
	selectedProvider string
	isUploading      bool
	cancelUpload     context.CancelFunc

	// Прогресс (потокобезопасный доступ)
	progressMutex  sync.RWMutex
	latestProgress *providers.UploadProgress
	totalSize      int64
	stopUIUpdate   chan struct{}
	uploadResult   chan *uploadCompletion
}

// uploadCompletion содержит результат загрузки
type uploadCompletion struct {
	result *providers.UploadResult
	err    error
}

// NewUploadTab создает новую вкладку загрузки
func NewUploadTab(app *App) *UploadTab {
	tab := &UploadTab{app: app,
		progressBinding:  binding.NewFloat(),
		uploadedBinding:  binding.NewString(),
		speedBinding:     binding.NewString(),
		etaBinding:       binding.NewString(),
		resultBinding:    binding.NewString(),
		uploadBtnBinding: binding.NewString(),
	}

	// Устанавливаем начальные значения
	tab.uploadedBinding.Set("")
	tab.speedBinding.Set("")
	tab.etaBinding.Set("")
	tab.resultBinding.Set("")
	tab.uploadBtnBinding.Set(localization.T("Start Upload"))

	return tab
}

// Build создает UI вкладки загрузки
func (t *UploadTab) Build() fyne.CanvasObject {
	// Выбор провайдера
	providerLabel := widget.NewLabel(localization.T("Select Providers"))
	t.providerSelect = widget.NewSelect([]string{}, func(selected string) {
		t.selectedProvider = selected
		t.updateUploadButton()
	})

	// Кнопка выбора файла
	t.filePathLabel = widget.NewLabel(localization.T("No file selected"))
	t.selectFileBtn = widget.NewButton(localization.T("Select File"), t.onSelectFile)

	// Progress bar и информация о прогрессе
	t.progressBar = widget.NewProgressBarWithData(t.progressBinding)
	t.progressBar.Hide()

	// Используем data binding для потокобезопасного обновления
	t.uploadedLabel = widget.NewLabelWithData(t.uploadedBinding)
	t.uploadedLabel.Hide()

	t.speedLabel = widget.NewLabelWithData(t.speedBinding)
	t.speedLabel.Hide()

	t.etaLabel = widget.NewLabelWithData(t.etaBinding)
	t.etaLabel.Hide()

	// Кнопка загрузки с binding
	t.uploadBtn = widget.NewButtonWithIcon("", nil, t.onUpload)
	t.uploadBtn.Disable()

	// Результат загрузки с binding
	t.resultLabel = widget.NewLabelWithData(t.resultBinding)
	t.resultLabel.Wrapping = fyne.TextWrapWord

	// Обновляем список провайдеров
	t.updateProviderList()

	// Обновляем текст кнопки из binding при старте
	go func() {
		text, _ := t.uploadBtnBinding.Get()
		t.uploadBtn.SetText(text)
	}()

	// Компоновка UI
	providerRow := container.NewBorder(nil, nil, providerLabel, nil, t.providerSelect)
	fileRow := container.NewBorder(nil, nil, nil, t.selectFileBtn, t.filePathLabel)

	progressGroup := container.NewVBox(
		t.progressBar,
		t.uploadedLabel,
		t.speedLabel,
		t.etaLabel,
	)

	content := container.NewVBox(
		widget.NewLabel(localization.T("Upload")),
		widget.NewSeparator(),
		providerRow,
		fileRow,
		widget.NewSeparator(),
		progressGroup,
		t.uploadBtn,
		widget.NewSeparator(),
		t.resultLabel,
	)

	return container.NewPadded(content)
}

// updateProviderList обновляет список доступных провайдеров
func (t *UploadTab) updateProviderList() {
	enabledProviders := t.app.GetEnabledProviders()
	providerNames := make([]string, 0, len(enabledProviders))

	for _, p := range enabledProviders {
		providerNames = append(providerNames, p.Name())
	}

	t.providerSelect.Options = providerNames

	if len(providerNames) > 0 && t.selectedProvider == "" {
		t.providerSelect.SetSelected(providerNames[0])
		t.selectedProvider = providerNames[0]
	}
}

// onSelectFile обработчик выбора файла
func (t *UploadTab) onSelectFile() {
	// Создаем file dialog вручную, чтобы установить custom размер
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			t.showFriendlyError(err)
			return
		}
		if reader == nil {
			return // Пользователь отменил
		}
		defer reader.Close()

		t.selectedFile = reader.URI()

		// Получаем размер файла
		fileInfo, err := os.Stat(reader.URI().Path())
		if err != nil {
			t.filePathLabel.SetText(fmt.Sprintf("Selected: %s", reader.URI().Name()))
		} else {
			sizeStr := providers.FormatSize(fileInfo.Size())
			t.filePathLabel.SetText(fmt.Sprintf("Selected: %s (%s)", reader.URI().Name(), sizeStr))
		}

		t.updateUploadButton()
	}, t.app.MainWindow())

	// Устанавливаем больший размер для удобства
	fileDialog.Resize(fyne.NewSize(800, 600))
	fileDialog.Show()
}

// onUpload обработчик загрузки файла
func (t *UploadTab) onUpload() {
	if t.selectedFile == nil || t.selectedProvider == "" {
		return
	}

	if t.isUploading {
		// Отмена загрузки
		if t.cancelUpload != nil {
			t.cancelUpload()
		}
		return
	}

	// Начинаем загрузку
	t.startUpload()
}

// startUpload начинает процесс загрузки
func (t *UploadTab) startUpload() {
	t.isUploading = true
	t.uploadBtn.SetText(localization.T("Cancel"))
	t.resultBinding.Set("")

	// Показываем элементы прогресса
	t.progressBar.Show()
	t.uploadedLabel.Show()
	t.speedLabel.Show()
	t.etaLabel.Show()

	// Используем binding для инициализации
	t.progressBinding.Set(0)
	t.uploadedBinding.Set(localization.T("Uploading..."))
	t.speedBinding.Set("Speed: calculating...")
	t.etaBinding.Set("ETA: calculating...")

	// Получаем провайдер
	provider, ok := t.app.GetProvider(t.selectedProvider)
	if !ok {
		t.finishUpload(fmt.Errorf("provider not found: %s", t.selectedProvider))
		return
	}

	// Открываем файл
	file, err := os.Open(t.selectedFile.Path())
	if err != nil {
		t.finishUpload(err)
		return
	}

	// Получаем размер файла
	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		t.finishUpload(err)
		return
	}

	fileSize := fileInfo.Size()
	filename := t.selectedFile.Name()

	// Создаем контекст с возможностью отмены
	ctx, cancel := context.WithCancel(context.Background())
	t.cancelUpload = cancel

	// Канал для прогресса
	progressChan := make(chan providers.UploadProgress, 10)

	// Канал для результата загрузки
	t.uploadResult = make(chan *uploadCompletion, 1)

	// Запускаем загрузку в горутине
	go func() {
		defer file.Close()
		defer close(progressChan)

		result, err := provider.Upload(ctx, file, filename, fileSize, progressChan)

		// Отправляем результат в канал вместо прямого вызова UI функций
		t.uploadResult <- &uploadCompletion{
			result: result,
			err:    err,
		}
	}()

	// Сохраняем размер файла для UI обновлений
	t.totalSize = fileSize

	// Создаем канал для остановки UI обновлений
	t.stopUIUpdate = make(chan struct{})

	// Запускаем горутину для обновления UI (потокобезопасно)
	go t.updateUIFromProgress()

	// Отслеживаем прогресс (сохраняем данные без обновления UI)
	go t.trackProgress(progressChan)
}

// trackProgress читает прогресс из канала и сохраняет его (БЕЗ обновления UI)
func (t *UploadTab) trackProgress(progressChan <-chan providers.UploadProgress) {
	for progress := range progressChan {
		// Потокобезопасно сохраняем последний прогресс
		t.progressMutex.Lock()
		progressCopy := progress
		t.latestProgress = &progressCopy
		t.progressMutex.Unlock()
	}
}

// updateUIFromProgress обновляет UI из тикера (потокобезопасно)
func (t *UploadTab) updateUIFromProgress() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopUIUpdate:
			return

		case completion := <-t.uploadResult:
			// Обрабатываем завершение загрузки из безопасного контекста
			t.finishUpload(completion.err)
			if completion.err == nil && completion.result != nil {
				t.showResult(completion.result)
			}
			return

		case <-ticker.C:
			// Потокобезопасно читаем последний прогресс
			t.progressMutex.RLock()
			progress := t.latestProgress
			totalSize := t.totalSize
			t.progressMutex.RUnlock()

			if progress == nil {
				continue
			}

			// Обновляем UI через data binding (ПОТОКОБЕЗОПАСНО!)
			percentage := float64(progress.Percentage) / 100.0
			t.progressBinding.Set(percentage)

			uploadedStr := providers.FormatSize(progress.BytesUploaded)
			totalStr := providers.FormatSize(totalSize)
			t.uploadedBinding.Set(fmt.Sprintf("Uploaded: %s / %s", uploadedStr, totalStr))

			speedStr := providers.FormatSpeed(progress.Speed)
			t.speedBinding.Set(fmt.Sprintf("Speed: %s", speedStr))

			bytesRemaining := totalSize - progress.BytesUploaded
			etaStr := providers.CalculateETA(bytesRemaining, progress.Speed)
			t.etaBinding.Set(fmt.Sprintf("ETA: %s", etaStr))
		}
	}
}

// finishUpload завершает процесс загрузки (вызывается из горутины!)
func (t *UploadTab) finishUpload(err error) {
	// Останавливаем UI обновления
	if t.stopUIUpdate != nil {
		close(t.stopUIUpdate)
		t.stopUIUpdate = nil
	}

	t.isUploading = false
	// НЕ вызываем SetText из горутины - будет ошибка!
	// t.uploadBtn.SetText("Upload")
	t.cancelUpload = nil

	// Сбрасываем прогресс
	t.progressMutex.Lock()
	t.latestProgress = nil
	t.progressMutex.Unlock()

	if err != nil {
		// Логируем ошибку с контекстом
		logging.ErrorWithError("Upload failed",
			err,
			"provider", t.selectedProvider,
			"filename", t.selectedFile.Name(),
			"filesize", t.totalSize,
		)

		// Отправляем уведомление об ошибке
		t.app.SendNotification(
			localization.T("Upload Failed"),
			fmt.Sprintf("%s - %s", t.selectedFile.Name(), localization.T("Check logs for details")),
		)

		// Показываем дружественное сообщение об ошибке
		t.showFriendlyError(err)
	}

	fyne.Do(func() {
		t.uploadBtn.SetText(localization.T("Start Upload"))
		t.progressBar.Hide()
		t.uploadedLabel.Hide()
		t.speedLabel.Hide()
		t.etaLabel.Hide()
	})
}

// showResult показывает результат загрузки (вызывается из горутины!)
func (t *UploadTab) showResult(result *providers.UploadResult) {
	if result == nil {
		return
	}

	// Отправляем уведомление об успехе
	t.app.SendNotification(
		localization.T("Upload Complete"),
		fmt.Sprintf("%s uploaded to %s", t.selectedFile.Name(), t.selectedProvider),
	)

	// Создаем контейнер для результатов
	content := container.NewVBox()

	// Добавляем сообщение об успехе
	successLabel := widget.NewLabel(localization.T("Upload Complete") + "!")
	successLabel.TextStyle = fyne.TextStyle{Bold: true}
	content.Add(successLabel)

	// Функция для создания строки с URL и кнопкой копирования
	createURLRow := func(label, url string) *fyne.Container {
		// Label для описания
		urlLabel := widget.NewLabel(label + ":")
		urlLabel.TextStyle = fyne.TextStyle{Bold: true}

		// Entry для URL (read-only, можно выделять текст)
		urlEntry := widget.NewLabel(url)
		urlEntry.SetText(url)
		urlEntry.Selectable = true
		//urlEntry.Disable() // Disable делает его read-only но позволяет выделять текст

		// Кнопка копирования
		copyBtn := widget.NewButton(localization.T("Copy"), func() {
			t.app.MainWindow().Clipboard().SetContent(url)
			// Можно добавить уведомление
			dialog.ShowInformation(localization.T("Copied to clipboard"), localization.T("Link copied"), t.app.MainWindow())
		})

		copyBtn.SetIcon(theme.ContentCopyIcon())

		return container.NewBorder(
			nil, nil,
			nil, copyBtn, // кнопка справа
			container.NewVBox(urlLabel, urlEntry),
		)
	}

	// Добавляем основной URL
	if result.URL != "" {
		content.Add(widget.NewLabel("")) // пустая строка для отступа
		content.Add(createURLRow("URL", result.URL))
	}

	// Добавляем Download URL если есть
	if result.DownloadURL != "" {
		content.Add(widget.NewLabel("")) // пустая строка для отступа
		content.Add(createURLRow("Download URL", result.DownloadURL))
	}

	// Добавляем Delete URL если есть
	if result.DeleteURL != "" {
		content.Add(widget.NewLabel("")) // пустая строка для отступа
		content.Add(createURLRow("Delete URL", result.DeleteURL))
	}

	// Добавляем сообщение если есть
	if result.Message != "" {
		content.Add(widget.NewLabel("")) // пустая строка для отступа
		messageLabel := widget.NewLabel(result.Message)
		messageLabel.Wrapping = fyne.TextWrapWord
		content.Add(messageLabel)
	}

	// Показываем кастомный диалог
	d := dialog.NewCustom(localization.T("Upload Results"), "Close", content, t.app.MainWindow())
	d.Resize(fyne.NewSize(600, 400))
	d.Show()
}

// updateUploadButton обновляет состояние кнопки загрузки
func (t *UploadTab) updateUploadButton() {
	if t.selectedFile != nil && t.selectedProvider != "" && !t.isUploading {
		t.uploadBtn.Enable()
	} else if !t.isUploading {
		t.uploadBtn.Disable()
	}
}

// Refresh обновляет список провайдеров (вызывается после изменения настроек)
func (t *UploadTab) Refresh() {
	t.updateProviderList()
	t.updateUploadButton()
}

// showFriendlyError показывает дружественное сообщение об ошибке
func (t *UploadTab) showFriendlyError(err error) {
	if err == nil {
		return
	}

	friendlyErr := MakeFriendly(err)
	message := FormatErrorMessage(friendlyErr)

	// Показываем custom dialog с понятным сообщением
	content := widget.NewLabel(message)
	content.Wrapping = fyne.TextWrapWord

	d := dialog.NewCustom(friendlyErr.Title, "OK", content, t.app.MainWindow())
	d.Resize(fyne.NewSize(500, 200))
	d.Show()
}
