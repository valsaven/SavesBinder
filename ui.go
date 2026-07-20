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

// wrapTextToWidth inserts newlines so text fits within maxW (using fyne.MeasureText).
// Prefers breaking at spaces; falls back to character wrap for long paths.
func wrapTextToWidth(text string, maxW, textSize float32, style fyne.TextStyle) string {
	if maxW < 40 {
		maxW = 40
	}
	if text == "" || fyne.MeasureText(text, textSize, style).Width <= maxW {
		return text
	}

	fits := func(s string) bool {
		return fyne.MeasureText(s, textSize, style).Width <= maxW
	}

	var lines []string
	var line []rune

	emit := func(s string) {
		if s != "" {
			lines = append(lines, s)
		}
	}

	for _, r := range text {
		if r == '\n' {
			emit(string(line))
			line = nil
			continue
		}

		trial := string(append(append([]rune{}, line...), r))
		if fits(trial) {
			line = append(line, r)
			continue
		}

		// Current line full — try soft wrap at last space
		if sp := lastSpaceIndex(line); sp >= 0 {
			emit(string(line[:sp]))
			line = append(append([]rune{}, line[sp+1:]...), r)
		} else if len(line) > 0 {
			emit(string(line))
			line = []rune{r}
		} else {
			// Single rune wider than maxW — still emit so we make progress
			emit(string(r))
			line = nil
			continue
		}

		// Soft-wrap rest may still exceed maxW (long token); hard-split it
		for len(line) > 0 && !fits(string(line)) {
			if len(line) == 1 {
				emit(string(line))
				line = nil
				break
			}
			// Largest prefix that fits
			fit := 1
			for fit < len(line) && fits(string(line[:fit+1])) {
				fit++
			}
			emit(string(line[:fit]))
			line = line[fit:]
		}
	}
	emit(string(line))

	if len(lines) == 0 {
		return text
	}
	return strings.Join(lines, "\n")
}

func lastSpaceIndex(runes []rune) int {
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == ' ' || runes[i] == '\t' {
			return i
		}
	}
	return -1
}

// showHoverTooltip shows text near anchor, fully inside the canvas (flips when near edges).
// Sizing is based on fyne.MeasureText + explicit newlines — not Label WordWrap MinSize,
// which is unreliable before the widget is laid out in a parent of known width.
func showHoverTooltip(canvas fyne.Canvas, text string, anchor fyne.Position) *widget.PopUp {
	if canvas == nil || text == "" {
		return nil
	}

	const edgePad float32 = 8
	const tipOffset float32 = 12

	canvasSize := canvas.Size()
	if canvasSize.Width < 1 || canvasSize.Height < 1 {
		// Fallback if canvas not laid out yet
		lbl := widget.NewLabel(text)
		popup := widget.NewPopUp(lbl, canvas)
		popup.ShowAtPosition(anchor)
		return popup
	}

	th := fyne.CurrentApp().Settings().Theme()
	textSize := th.Size(theme.SizeNameText)
	style := fyne.TextStyle{}
	// Leave room for PopUp chrome / padding around the label
	innerPad := th.Size(theme.SizeNameInnerPadding) * 2

	maxContentW := canvasSize.Width - edgePad*2 - innerPad
	if maxContentW > 360 {
		maxContentW = 360
	}
	if maxContentW < 120 {
		maxContentW = 120
	}

	wrapped := wrapTextToWidth(text, maxContentW, textSize, style)
	lbl := widget.NewLabel(wrapped)
	// Explicit newlines already encode wrapping; keep wrap off so MinSize is stable.
	lbl.Wrapping = fyne.TextWrapOff

	popup := widget.NewPopUp(lbl, canvas)
	size := popup.MinSize()

	// Cap to canvas (safety net)
	maxPopW := canvasSize.Width - edgePad*2
	maxPopH := canvasSize.Height - edgePad*2
	if size.Width > maxPopW {
		size.Width = maxPopW
	}
	if size.Height > maxPopH {
		size.Height = maxPopH
	}

	// Prefer below-right of the cursor; flip when that would leave the window.
	x := anchor.X + tipOffset
	y := anchor.Y + tipOffset

	if x+size.Width > canvasSize.Width-edgePad {
		x = anchor.X - size.Width - tipOffset
	}
	if y+size.Height > canvasSize.Height-edgePad {
		y = anchor.Y - size.Height - tipOffset
	}

	// Hard clamp so the whole popup stays on-canvas (right-side Delete button case).
	if x < edgePad {
		x = edgePad
	}
	if y < edgePad {
		y = edgePad
	}
	if x+size.Width > canvasSize.Width-edgePad {
		x = canvasSize.Width - size.Width - edgePad
		if x < edgePad {
			x = edgePad
		}
	}
	if y+size.Height > canvasSize.Height-edgePad {
		y = canvasSize.Height - size.Height - edgePad
		if y < edgePad {
			y = edgePad
		}
	}

	popup.Resize(size)
	popup.ShowAtPosition(fyne.NewPos(x, y))
	// Re-assert size after Show (overlay layout can reset to a broken MinSize).
	popup.Resize(size)
	popup.Move(fyne.NewPos(x, y))
	return popup
}

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

	// Debounce show; UI mutations must run on the Fyne thread
	go func() {
		time.Sleep(500 * time.Millisecond)
		fyne.Do(func() {
			if !h.hovered || h.popup != nil || h.hint == "" || h.canvas == nil {
				return
			}
			h.popup = showHoverTooltip(h.canvas, h.hint, h.currentPos)
		})
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

// displayPair is one filtered list entry with path health labels
type displayPair struct {
	Pair         LinkPair
	TargetStatus string
	LinkStatus   string
}

// HoverLabel is a single-line label that shows the full text in a tooltip on hover.
// Used for long paths so list rows stay fixed-height while remaining readable.
type HoverLabel struct {
	widget.Label
	fullText   string
	canvas     fyne.Canvas
	popup      *widget.PopUp
	hovered    bool
	currentPos fyne.Position
}

func NewHoverLabel(canvas fyne.Canvas) *HoverLabel {
	l := &HoverLabel{canvas: canvas}
	l.Wrapping = fyne.TextWrapOff
	l.Truncation = fyne.TextTruncateEllipsis
	l.ExtendBaseWidget(l)
	return l
}

// SetFullText updates both the truncated display text and the hover tooltip content.
func (l *HoverLabel) SetFullText(text string) {
	l.fullText = text
	l.SetText(text)
}

func (l *HoverLabel) MouseIn(e *desktop.MouseEvent) {
	l.hovered = true
	l.currentPos = e.AbsolutePosition
	if l.fullText == "" || l.canvas == nil {
		return
	}
	go func() {
		time.Sleep(400 * time.Millisecond)
		fyne.Do(func() {
			if !l.hovered || l.popup != nil || l.fullText == "" || l.canvas == nil {
				return
			}
			l.popup = showHoverTooltip(l.canvas, l.fullText, l.currentPos)
		})
	}()
}

func (l *HoverLabel) MouseOut() {
	l.hovered = false
	if l.popup != nil {
		l.popup.Hide()
		l.popup = nil
	}
}

func (l *HoverLabel) MouseMoved(e *desktop.MouseEvent) {
	l.currentPos = e.AbsolutePosition
}

// pairRow is a reusable virtualized list row for a bound save pair
type pairRow struct {
	widget.BaseWidget
	sourceLabel *HoverLabel
	targetLabel *HoverLabel
	btnRestore  *HoverButton
	btnUnbind   *HoverButton
	btnDestroy  *HoverButton
}

func newPairRow(canvas fyne.Canvas) *pairRow {
	r := &pairRow{
		sourceLabel: NewHoverLabel(canvas),
		targetLabel: NewHoverLabel(canvas),
	}

	r.btnRestore = NewHoverButton(T("btn_restore"), theme.HistoryIcon(), T("hint_restore"), canvas, nil)
	r.btnRestore.Importance = widget.HighImportance

	r.btnUnbind = NewHoverButton(T("btn_unbind"), theme.ContentRemoveIcon(), T("hint_unbind"), canvas, nil)

	r.btnDestroy = NewHoverButton(T("btn_destroy"), theme.DeleteIcon(), T("hint_destroy"), canvas, nil)
	r.btnDestroy.Importance = widget.DangerImportance

	r.ExtendBaseWidget(r)
	return r
}

func (r *pairRow) CreateRenderer() fyne.WidgetRenderer {
	pathBlock := container.NewVBox(r.sourceLabel, r.targetLabel)
	actionGroup := container.NewHBox(r.btnRestore, r.btnUnbind, r.btnDestroy)
	// Theme padding instead of dummy space Labels (cheaper layout, no extra widgets)
	row := container.NewPadded(container.NewBorder(nil, nil, nil, actionGroup, pathBlock))
	return widget.NewSimpleRenderer(row)
}

func (r *pairRow) bind(p displayPair, a *SavesBinderApp) {
	r.sourceLabel.SetFullText(fmt.Sprintf("source: \"%s\"%s", p.Pair.Target, p.TargetStatus))
	r.targetLabel.SetFullText(fmt.Sprintf("target: \"%s\"%s", p.Pair.Link, p.LinkStatus))

	pair := p.Pair
	r.btnRestore.OnTapped = func() {
		a.restoreSave(pair)
	}
	r.btnUnbind.OnTapped = func() {
		a.unbindSave(pair)
	}
	r.btnDestroy.OnTapped = func() {
		a.destroySave(pair)
	}
}

func pairCacheKey(p LinkPair) string {
	return p.Target + "\x00" + p.Link
}

// refreshList rebuilds the filtered view from the in-memory cache (no FS I/O)
// and kicks off a background status scan when needed.
func (a *SavesBinderApp) refreshList() {
	a.refreshListFromCache()
	a.scheduleStatusScan()
}

// refreshListFromCache filters pairs and applies cached health labels only.
func (a *SavesBinderApp) refreshListFromCache() {
	filter := ""
	if a.searchEntry != nil {
		filter = strings.ToLower(strings.TrimSpace(a.searchEntry.Text))
	}

	// Reuse backing array to avoid reallocating on every refresh
	a.filteredPairs = a.filteredPairs[:0]

	a.statusMu.RLock()
	cache := a.pathHealthCache
	for _, pair := range a.db.Links {
		if filter != "" {
			targetLower := strings.ToLower(pair.Target)
			linkLower := strings.ToLower(pair.Link)

			if !strings.Contains(targetLower, filter) && !strings.Contains(linkLower, filter) {
				continue
			}
		}

		dp := displayPair{Pair: pair}
		if h, ok := cache[pairCacheKey(pair)]; ok && h.Checked {
			if h.TargetMissing {
				dp.TargetStatus = T("orig_deleted")
			}
			if h.LinkMissing {
				dp.LinkStatus = T("link_broken")
			}
		}
		a.filteredPairs = append(a.filteredPairs, dp)
	}
	a.statusMu.RUnlock()

	if a.pairList != nil {
		a.pairList.Refresh()
	}
}

// scheduleStatusScan checks path existence off the UI thread and refreshes labels.
func (a *SavesBinderApp) scheduleStatusScan() {
	a.statusMu.Lock()
	if a.pathHealthCache == nil {
		a.pathHealthCache = make(map[string]pathHealth)
	}
	a.statusEpoch++
	epoch := a.statusEpoch
	// Snapshot pairs so we don't race with bind/unbind while scanning
	pairs := make([]LinkPair, len(a.db.Links))
	copy(pairs, a.db.Links)
	a.statusMu.Unlock()

	go func(epoch uint64, pairs []LinkPair) {
		updated := make(map[string]pathHealth, len(pairs))
		for _, pair := range pairs {
			_, errTarget := os.Stat(pair.Target)
			_, errLink := os.Stat(pair.Link)
			updated[pairCacheKey(pair)] = pathHealth{
				TargetMissing: os.IsNotExist(errTarget),
				LinkMissing:   os.IsNotExist(errLink),
				Checked:       true,
			}
		}

		a.statusMu.Lock()
		// Drop results if a newer scan was scheduled (stale)
		if epoch != a.statusEpoch {
			a.statusMu.Unlock()
			return
		}
		a.pathHealthCache = updated
		a.statusMu.Unlock()

		fyne.Do(func() {
			a.refreshListFromCache()
		})
	}(epoch, pairs)
}

// invalidatePathHealth clears cached status (e.g. after bind/unbind/restore).
func (a *SavesBinderApp) invalidatePathHealth() {
	a.statusMu.Lock()
	a.pathHealthCache = make(map[string]pathHealth)
	a.statusEpoch++
	a.statusMu.Unlock()
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
				a.invalidatePathHealth()
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
		// Debounce: rebuild filtered list only after typing pauses (~200ms)
		a.searchMu.Lock()
		if a.searchTimer != nil {
			a.searchTimer.Stop()
		}
		a.searchTimer = time.AfterFunc(200*time.Millisecond, func() {
			fyne.Do(func() {
				a.refreshList()
			})
		})
		a.searchMu.Unlock()
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

	// Virtualized list: only visible rows are rendered/updated
	a.pairList = widget.NewList(
		func() int {
			return len(a.filteredPairs)
		},
		func() fyne.CanvasObject {
			return newPairRow(a.window.Canvas())
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			row, ok := obj.(*pairRow)
			if !ok || id < 0 || id >= len(a.filteredPairs) {
				return
			}
			row.bind(a.filteredPairs[id], a)
		},
	)

	a.refreshList()

	// Form (with all inputs including Search) at the bottom, list above it
	return container.NewBorder(nil, bottomFrame, nil, nil, a.pairList)
}
