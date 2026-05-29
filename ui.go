package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// HoverButton wraps a standard Fyne Button to show a floating tooltip on hover
type HoverButton struct {
	widget.Button
	hint       string
	canvas     fyne.Canvas
	popup      *widget.PopUp
	hovered    bool
	currentPos fyne.Position
}

// NewHoverButton creates a new custom button with tooltip support
func NewHoverButton(text string, icon fyne.Resource, hint string, canvas fyne.Canvas, tapped func()) *HoverButton {
	b := &HoverButton{hint: hint, canvas: canvas}
	b.Text = text
	b.Icon = icon
	b.OnTapped = tapped
	b.ExtendBaseWidget(b) // Required to make custom Fyne widgets work
	return b
}

// MouseIn is called when the desktop mouse enters the widget
func (h *HoverButton) MouseIn(e *desktop.MouseEvent) {
	h.Button.MouseIn(e)
	h.hovered = true
	h.currentPos = e.AbsolutePosition

	// Start a debounce timer in a separate goroutine
	go func() {
		time.Sleep(500 * time.Millisecond) // 500ms delay before showing

		// Check if mouse is still hovering and popup isn't already shown
		if h.hovered && h.popup == nil && h.hint != "" && h.canvas != nil {
			lbl := widget.NewLabel(h.hint)

			// Use a safe offset (20px) from the LAST recorded mouse position
			safePos := fyne.NewPos(h.currentPos.X+20, h.currentPos.Y+20)

			h.popup = widget.NewPopUp(lbl, h.canvas)
			h.popup.ShowAtPosition(safePos)
		}
	}()
}

// MouseOut is called when the desktop mouse leaves the widget
func (h *HoverButton) MouseOut() {
	h.Button.MouseOut()
	h.hovered = false // Cancel the pending tooltip if it hasn't shown yet

	if h.popup != nil {
		h.popup.Hide()
		h.popup = nil
	}
}

// MouseMoved is called when the desktop mouse moves over the widget
func (h *HoverButton) MouseMoved(e *desktop.MouseEvent) {
	h.Button.MouseMoved(e)
	h.currentPos = e.AbsolutePosition // Constantly update position while moving
}

// refreshList rebuilds the UI list and checks the status of paths
func (a *SavesBinderApp) refreshList() {
	a.listContainer.Objects = nil

	filter := ""
	if a.searchEntry != nil {
		filter = strings.ToLower(strings.TrimSpace(a.searchEntry.Text))
	}

	for _, pair := range a.db.Links {
		if filter != "" {
			targetLower := strings.ToLower(pair.Target)
			linkLower := strings.ToLower(pair.Link)

			if !strings.Contains(targetLower, filter) && !strings.Contains(linkLower, filter) {
				continue
			}
		}

		p := pair // Capture variable for closure

		_, errTarget := os.Stat(p.Target)
		_, errLink := os.Stat(p.Link)

		targetStatus := ""
		linkStatus := ""
		if os.IsNotExist(errTarget) {
			targetStatus = T("orig_deleted")
		}
		if os.IsNotExist(errLink) {
			linkStatus = T("link_broken")
		}

		// Create two path strings
		sourceLabel := widget.NewLabel(fmt.Sprintf("source: \"%s\"%s", p.Target, targetStatus))
		targetLabel := widget.NewLabel(fmt.Sprintf("target: \"%s\"%s", p.Link, linkStatus))

		// Enable automatic line breaks to prevent long paths from breaking the layout
		sourceLabel.Wrapping = fyne.TextWrapBreak
		targetLabel.Wrapping = fyne.TextWrapBreak

		// Make the source text slightly dimmer or leave it as is,
		// but combine them into a vertical block
		pathBlock := container.NewVBox(sourceLabel, targetLabel)

		// Create symmetrical padding from spaces
		leftPadding := widget.NewLabel("  ")
		rightPadding := widget.NewLabel("  ")

		// Create the 3 action buttons using our custom HoverButton
		btnRestore := NewHoverButton(T("btn_restore"), theme.HistoryIcon(), T("hint_restore"), a.window.Canvas(), func() {
			a.restoreSave(p)
		})
		btnRestore.Importance = widget.HighImportance // Blue/Primary color

		btnUnbind := NewHoverButton(T("btn_unbind"), theme.ContentRemoveIcon(), T("hint_unbind"), a.window.Canvas(), func() {
			a.unbindSave(p)
		})
		// Default importance (neutral)

		btnDestroy := NewHoverButton(T("btn_destroy"), theme.DeleteIcon(), T("hint_destroy"), a.window.Canvas(), func() {
			a.destroySave(p)
		})
		btnDestroy.Importance = widget.DangerImportance // Red color

		// Combine buttons into a horizontal group AND add rightPadding to fix the scrollbar overlap
		actionGroup := container.NewHBox(btnRestore, btnUnbind, btnDestroy, rightPadding)

		// Assemble the row: padding on the left, paths in center, buttons + padding on the right
		row := container.NewBorder(nil, nil, leftPadding, actionGroup, pathBlock)

		a.listContainer.Add(container.NewVBox(row, widget.NewSeparator()))
	}

	a.listContainer.Refresh()
}

// buildUI constructs the interface and populates data bindings
func (a *SavesBinderApp) buildUI() fyne.CanvasObject {
	prefs := a.app.Preferences()

	savedDbPath := a.getDbPath()
	savedStorage := prefs.StringWithFallback("central_storage", "")
	confirmDelete := prefs.BoolWithFallback("confirm_delete", true)

	// --- Database file selection field ---
	a.dbPathEntry = widget.NewEntry()
	a.dbPathEntry.SetText(savedDbPath)

	btnBrowseDb := widget.NewButton("Browse...", func() {
		// Open a dialog to select an existing file (or create a new .json)
		d := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) {
			if err == nil && file != nil {
				newPath := file.URI().Path()
				a.dbPathEntry.SetText(newPath)
				prefs.SetString("db_file_path", newPath)

				// Switch the database on the fly
				a.loadData()
				a.refreshList()
			}
		}, a.window)

		// Set a human-readable size and call it
		d.Resize(fyne.NewSize(800, 500))
		d.Show()
	})

	a.storageEntry = widget.NewEntry()
	a.storageEntry.SetPlaceHolder("Example: F:\\saves")
	a.storageEntry.SetText(savedStorage)

	btnBrowseStorage := widget.NewButton("Browse...", func() {
		d := dialog.NewFolderOpen(func(list fyne.ListableURI, err error) {
			if err == nil && list != nil {
				a.storageEntry.SetText(list.Path())
				prefs.SetString("central_storage", list.Path())
			}
		}, a.window)

		// Set a human-readable size and call it
		d.Resize(fyne.NewSize(800, 500))
		d.Show()
	})

	a.toBindEntry = widget.NewEntry()
	a.toBindEntry.SetPlaceHolder("Example: D:\\Documents\\American Truck Simulator")

	btnBrowseBind := widget.NewButton("Browse...", func() {
		d := dialog.NewFolderOpen(func(list fyne.ListableURI, err error) {
			if err == nil && list != nil {
				a.toBindEntry.SetText(list.Path())
			}
		}, a.window)

		// Set a human-readable size and call it
		d.Resize(fyne.NewSize(800, 500))
		d.Show()
	})

	a.confirmCheck = widget.NewCheck(T("check_confirm"), func(checked bool) {
		prefs.SetBool("confirm_delete", checked)
	})
	a.confirmCheck.SetChecked(confirmDelete)

	// --- Language Selector Setup ---
	currentLang := prefs.StringWithFallback("app_language", lang.currentLang)
	langSelect := widget.NewSelect([]string{"EN", "RU"}, func(selected string) {
		var code string
		if selected == "RU" {
			code = "ru"
		} else {
			code = "en"
		}

		if code != lang.currentLang {
			prefs.SetString("app_language", code)
			SetLanguage(code)
			// Re-render whole window content to apply new translations instantly
			a.window.SetContent(a.buildUI())
		}
	})

	if currentLang == "ru" {
		langSelect.SetSelected("RU")
	} else {
		langSelect.SetSelected("EN")
	}

	// Search input (filter for the list above). Placed inside the form so all inputs live together.
	a.searchEntry = widget.NewEntry()
	a.searchEntry.SetPlaceHolder(T("search_placeholder"))
	a.searchEntry.OnChanged = func(string) {
		a.refreshList()
	}

	form := widget.NewForm(
		widget.NewFormItem(T("lbl_search"), a.searchEntry),
		widget.NewFormItem(T("lbl_db_file"), container.NewBorder(nil, nil, nil, btnBrowseDb, a.dbPathEntry)),
		widget.NewFormItem(T("lbl_storage"), container.NewBorder(nil, nil, nil, btnBrowseStorage, a.storageEntry)),
		widget.NewFormItem(T("lbl_saves_path"), container.NewBorder(nil, nil, nil, btnBrowseBind, a.toBindEntry)),
	)

	btnBind := widget.NewButton(T("btn_bind"), func() {
		prefs.SetString("central_storage", a.storageEntry.Text)
		a.bindSave()
	})

	btnVerify := widget.NewButton(T("btn_verify"), func() {
		a.verifyLinks()
	})

	// Layout buttons on the left, check and lang selector on the right
	leftControls := container.NewHBox(btnBind, btnVerify)
	rightControls := container.NewHBox(a.confirmCheck, langSelect)

	// Border container puts left controls on Left, right controls on Right
	actions := container.NewBorder(nil, nil, leftControls, rightControls)

	bottomFrame := container.NewVBox(widget.NewSeparator(), form, actions)

	// List container for link pairs
	a.listContainer = container.NewVBox()
	scrollContainer := container.NewScroll(a.listContainer)

	a.refreshList()

	// Form (with all inputs including Search) at the bottom, list above it
	return container.NewBorder(nil, bottomFrame, nil, nil, scrollContainer)
}
