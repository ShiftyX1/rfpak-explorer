package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var version = "dev"

type PakExplorer struct {
	window       fyne.Window
	reader       *PakReader
	tree         *widget.Tree
	searchEntry  *widget.Entry
	infoText     *widget.RichText
	previewStack *fyne.Container
	textPreview  *widget.Entry
	imagePreview *canvas.Image
	imageInfo    *widget.Label
	statusBar    *widget.Label

	treeData     map[string][]string
	fileEntries  map[string]*FileEntry
	currentEntry *FileEntry
	filteredData map[string][]string
}

func NewPakExplorer() *PakExplorer {
	app := app.NewWithID("com.rayflow.pakexplorer")
	window := app.NewWindow("RayFlow PAK Explorer")
	window.Resize(fyne.NewSize(1200, 700))

	explorer := &PakExplorer{
		window:      window,
		reader:      NewPakReader(),
		treeData:    make(map[string][]string),
		fileEntries: make(map[string]*FileEntry),
	}

	explorer.setupUI()
	return explorer
}

func (e *PakExplorer) setupUI() {
	openItem := fyne.NewMenuItem("Open Archive...", e.openArchive)
	openItem.Icon = theme.FolderOpenIcon()

	closeItem := fyne.NewMenuItem("Close Archive", e.closeArchive)

	quitItem := fyne.NewMenuItem("Quit", func() {
		e.window.Close()
	})

	fileMenu := fyne.NewMenu("File", openItem, closeItem, fyne.NewMenuItemSeparator(), quitItem)

	aboutItem := fyne.NewMenuItem("About", e.showAbout)
	helpMenu := fyne.NewMenu("Help", aboutItem)

	mainMenu := fyne.NewMainMenu(fileMenu, helpMenu)
	e.window.SetMainMenu(mainMenu)

	openBtn := widget.NewButtonWithIcon("Open Archive", theme.FolderOpenIcon(), e.openArchive)
	extractBtn := widget.NewButtonWithIcon("Extract Selected", theme.DownloadIcon(), e.extractSelected)
	extractAllBtn := widget.NewButtonWithIcon("Extract All", theme.DownloadIcon(), e.extractAll)

	searchLabel := widget.NewLabel("Search:")
	e.searchEntry = widget.NewEntry()
	e.searchEntry.SetPlaceHolder("Filter files...")
	e.searchEntry.OnChanged = e.filterFiles

	toolbar := container.NewBorder(nil, nil, nil,
		container.NewHBox(searchLabel, e.searchEntry),
		container.NewHBox(openBtn, extractBtn, extractAllBtn),
	)

	e.tree = widget.NewTree(
		e.childUIDs,
		e.isBranch,
		e.createTreeItem,
		e.updateTreeItem,
	)
	e.tree.OnSelected = e.onTreeSelected

	treeScroll := container.NewScroll(e.tree)
	treeScroll.SetMinSize(fyne.NewSize(400, 400))

	infoLabel := widget.NewLabelWithStyle("File Information", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	e.infoText = widget.NewRichTextFromMarkdown("")
	e.infoText.Wrapping = fyne.TextWrapWord

	previewLabel := widget.NewLabelWithStyle("Preview", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	e.textPreview = widget.NewMultiLineEntry()
	e.textPreview.Disable()

	e.imagePreview = canvas.NewImageFromImage(nil)
	e.imagePreview.FillMode = canvas.ImageFillContain
	e.imagePreview.SetMinSize(fyne.NewSize(400, 400))

	e.imageInfo = widget.NewLabel("")
	imageContainer := container.NewBorder(nil, e.imageInfo, nil, nil,
		container.NewScroll(e.imagePreview),
	)

	e.previewStack = container.NewStack(
		e.textPreview,
		imageContainer,
	)

	infoPanel := container.NewBorder(
		container.NewVBox(infoLabel, e.infoText, previewLabel),
		nil, nil, nil,
		e.previewStack,
	)

	split := container.NewHSplit(treeScroll, infoPanel)
	split.SetOffset(0.4)

	e.statusBar = widget.NewLabel("Ready")

	content := container.NewBorder(toolbar, e.statusBar, nil, nil, split)
	e.window.SetContent(content)
}

func (e *PakExplorer) openArchive() {
	dialog.ShowFileOpen(func(uc fyne.URIReadCloser, err error) {
		if err != nil || uc == nil {
			return
		}
		defer uc.Close()

		path := uc.URI().Path()
		if err := e.reader.Open(path); err != nil {
			dialog.ShowError(fmt.Errorf("Failed to open archive:\n%v", err), e.window)
			return
		}

		e.loadArchiveContents()
		e.statusBar.SetText(fmt.Sprintf("Loaded: %s (%d files)",
			filepath.Base(path), len(e.reader.Entries)))
	}, e.window)
}

func (e *PakExplorer) closeArchive() {
	e.reader.Close()
	e.treeData = make(map[string][]string)
	e.fileEntries = make(map[string]*FileEntry)
	e.filteredData = nil
	e.currentEntry = nil
	e.tree.Refresh()
	e.infoText.ParseMarkdown("")
	e.textPreview.SetText("")
	e.showTextPreview()
	e.statusBar.SetText("Archive closed")
}

func (e *PakExplorer) loadArchiveContents() {
	e.treeData = make(map[string][]string)
	e.fileEntries = make(map[string]*FileEntry)
	e.filteredData = nil

	for i := range e.reader.Entries {
		entry := &e.reader.Entries[i]
		parts := strings.Split(entry.FullPath, "/")

		parentPath := ""
		for j := range parts {
			currentPath := strings.Join(parts[:j+1], "/")

			if j < len(parts)-1 {
				if _, exists := e.treeData[parentPath]; !exists {
					e.treeData[parentPath] = []string{}
				}
				if !contains(e.treeData[parentPath], currentPath) {
					e.treeData[parentPath] = append(e.treeData[parentPath], currentPath)
				}
			} else {
				if _, exists := e.treeData[parentPath]; !exists {
					e.treeData[parentPath] = []string{}
				}
				e.treeData[parentPath] = append(e.treeData[parentPath], currentPath)
				e.fileEntries[currentPath] = entry
			}

			parentPath = currentPath
		}
	}

	e.tree.Refresh()
}

func (e *PakExplorer) filterFiles(search string) {
	if search == "" {
		e.filteredData = nil
		e.tree.Refresh()
		return
	}

	search = strings.ToLower(search)
	e.filteredData = make(map[string][]string)

	for path, entry := range e.fileEntries {
		if strings.Contains(strings.ToLower(entry.Name), search) ||
			strings.Contains(strings.ToLower(entry.FullPath), search) {
			parts := strings.Split(path, "/")
			parentPath := ""

			for j := 0; j < len(parts); j++ {
				currentPath := strings.Join(parts[:j+1], "/")

				if _, exists := e.filteredData[parentPath]; !exists {
					e.filteredData[parentPath] = []string{}
				}
				if !contains(e.filteredData[parentPath], currentPath) {
					e.filteredData[parentPath] = append(e.filteredData[parentPath], currentPath)
				}

				parentPath = currentPath
			}
		}
	}

	e.tree.Refresh()
}

func (e *PakExplorer) childUIDs(uid string) []string {
	data := e.treeData
	if e.filteredData != nil {
		data = e.filteredData
	}
	return data[uid]
}

func (e *PakExplorer) isBranch(uid string) bool {
	_, isFile := e.fileEntries[uid]
	return !isFile
}

func (e *PakExplorer) createTreeItem(branch bool) fyne.CanvasObject {
	return container.NewHBox(
		widget.NewIcon(theme.DocumentIcon()),
		widget.NewLabel(""),
	)
}

func (e *PakExplorer) updateTreeItem(uid string, branch bool, obj fyne.CanvasObject) {
	box := obj.(*fyne.Container)
	icon := box.Objects[0].(*widget.Icon)
	label := box.Objects[1].(*widget.Label)

	parts := strings.Split(uid, "/")
	name := parts[len(parts)-1]

	if branch {
		icon.SetResource(theme.FolderIcon())
		label.SetText(name + "/")
	} else {
		icon.SetResource(theme.DocumentIcon())
		label.SetText(name)
	}
}

func (e *PakExplorer) onTreeSelected(uid string) {
	entry, exists := e.fileEntries[uid]
	if !exists {
		e.currentEntry = nil
		e.infoText.ParseMarkdown("")
		e.textPreview.SetText("")
		e.showTextPreview()
		return
	}

	e.currentEntry = entry
	e.showFileInfo(entry)
}

func (e *PakExplorer) showFileInfo(entry *FileEntry) {
	info := fmt.Sprintf(`**Name:** %s  
**Path:** %s  
**Size:** %s  
**Offset:** %d bytes`,
		entry.Name,
		entry.FullPath,
		formatSize(entry.Size),
		entry.Offset,
	)

	e.infoText.ParseMarkdown(info)
	e.showFilePreview(entry)
}

func (e *PakExplorer) showFilePreview(entry *FileEntry) {
	data, err := e.reader.Extract(entry)
	if err != nil {
		e.textPreview.SetText("Failed to extract file.")
		e.showTextPreview()
		return
	}

	ext := strings.ToLower(filepath.Ext(entry.Name))
	if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".bmp" {
		e.showImagePreview(data, entry)
		return
	}

	if entry.Size > 1024*1024 {
		e.textPreview.SetText(fmt.Sprintf("File too large for preview (%s)", formatSize(entry.Size)))
		e.showTextPreview()
		return
	}

	text := string(data)
	if isPrintable(text) {
		if len(text) > 10000 {
			text = text[:10000] + "\n\n[... truncated ...]"
		}
		e.textPreview.SetText(text)
		e.showTextPreview()
	} else {
		hex := hexDump(data, 512)
		e.textPreview.SetText(hex)
		e.showTextPreview()
	}
}

func (e *PakExplorer) showImagePreview(data []byte, entry *FileEntry) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		e.textPreview.SetText("Failed to load image.")
		e.showTextPreview()
		return
	}

	e.imagePreview.Image = img
	e.imagePreview.Refresh()

	bounds := img.Bounds()
	info := fmt.Sprintf("%d × %d pixels • %s",
		bounds.Dx(), bounds.Dy(), formatSize(entry.Size))
	e.imageInfo.SetText(info)

	e.previewStack.Objects[0].Hide()
	e.previewStack.Objects[1].Show()
}

func (e *PakExplorer) showTextPreview() {
	e.previewStack.Objects[1].Hide()
	e.previewStack.Objects[0].Show()
}

func (e *PakExplorer) extractSelected() {
	if e.currentEntry == nil {
		return
	}

	dialog.ShowFileSave(func(uc fyne.URIWriteCloser, err error) {
		if err != nil || uc == nil {
			return
		}
		defer uc.Close()

		data, err := e.reader.Extract(e.currentEntry)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Failed to extract file:\n%v", err), e.window)
			return
		}

		if _, err := uc.Write(data); err != nil {
			dialog.ShowError(fmt.Errorf("Failed to save file:\n%v", err), e.window)
			return
		}

		dialog.ShowInformation("Success",
			fmt.Sprintf("File extracted successfully:\n%s", uc.URI().Path()),
			e.window)
	}, e.window)
}

func (e *PakExplorer) extractAll() {
	if len(e.reader.Entries) == 0 {
		return
	}

	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}

		outputPath := uri.Path()
		successCount := 0
		errorCount := 0

		for i := range e.reader.Entries {
			entry := &e.reader.Entries[i]
			data, err := e.reader.Extract(entry)
			if err != nil {
				errorCount++
				continue
			}

			filePath := filepath.Join(outputPath, entry.FullPath)
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				errorCount++
				continue
			}

			if err := os.WriteFile(filePath, data, 0644); err != nil {
				errorCount++
				continue
			}

			successCount++
		}

		msg := fmt.Sprintf("Extraction complete!\n\nSuccessful: %d", successCount)
		if errorCount > 0 {
			msg += fmt.Sprintf("\nErrors: %d", errorCount)
		}

		dialog.ShowInformation("Extraction Complete", msg, e.window)
		e.statusBar.SetText(fmt.Sprintf("Extracted %d files to %s", successCount, outputPath))
	}, e.window)
}

func (e *PakExplorer) showAbout() {
	about := widget.NewLabel(fmt.Sprintf(`RayFlow PAK Explorer
Version: %s

A graphical tool for browsing and extracting RFPK archive files.

Part of the RayFlow voxel game project.

Format: RFPK v1
Magic: 0x4B504652 ('RFPK')`, version))

	dialog.ShowCustom("About RayFlow PAK Explorer", "Close", about, e.window)
}

func (e *PakExplorer) Run() {
	if len(os.Args) > 1 {
		path := os.Args[1]
		if err := e.reader.Open(path); err == nil {
			e.loadArchiveContents()
			e.statusBar.SetText(fmt.Sprintf("Loaded: %s (%d files)",
				filepath.Base(path), len(e.reader.Entries)))
		}
	}

	e.window.ShowAndRun()
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func formatSize(size uint64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	s := float64(size)
	i := 0

	for s >= 1024.0 && i < len(units)-1 {
		s /= 1024.0
		i++
	}

	return fmt.Sprintf("%.1f %s", s, units[i])
}

func isPrintable(s string) bool {
	for _, r := range s {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
		if r > 126 && r < 160 {
			return false
		}
	}
	return true
}

func hexDump(data []byte, maxBytes int) string {
	if len(data) > maxBytes {
		data = data[:maxBytes]
	}

	var sb strings.Builder
	for i := 0; i < len(data); i += 16 {
		sb.WriteString(fmt.Sprintf("%08X  ", i))

		for j := 0; j < 16; j++ {
			if i+j < len(data) {
				sb.WriteString(fmt.Sprintf("%02X ", data[i+j]))
			} else {
				sb.WriteString("   ")
			}
		}

		sb.WriteString(" ")

		for j := 0; j < 16 && i+j < len(data); j++ {
			b := data[i+j]
			if b >= 32 && b < 127 {
				sb.WriteByte(b)
			} else {
				sb.WriteByte('.')
			}
		}

		sb.WriteString("\n")
	}

	if len(data) >= maxBytes {
		sb.WriteString("\n[... truncated ...]")
	}

	return sb.String()
}

func main() {
	explorer := NewPakExplorer()
	explorer.Run()
}
