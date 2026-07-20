package main

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
)

// SavesBinderApp holds references to UI components and application state
type SavesBinderApp struct {
	app           fyne.App
	window        fyne.Window
	dbPathEntry   *widget.Entry
	storageEntry  *widget.Entry
	toBindEntry   *widget.Entry
	confirmCheck  *widget.Check
	pairList      *widget.List
	searchEntry   *widget.Entry
	db            AppDatabase
	filteredPairs []displayPair

	searchMu    sync.Mutex
	searchTimer *time.Timer

	// pathHealthCache avoids synchronous os.Stat storms on every list refresh.
	// Keyed by target+"\x00"+link; updated asynchronously.
	statusMu        sync.RWMutex
	pathHealthCache map[string]pathHealth
	statusEpoch     uint64 // bumped to cancel outdated background scans
}

// pathHealth is the cached existence status of a bound pair's paths.
type pathHealth struct {
	TargetMissing bool
	LinkMissing   bool
	Checked       bool
}

func main() {
	initLang()

	myApp := app.NewWithID("com.github.valsaven.savesbinder")

	// Set app icon
	if resourceFloppyIconPng != nil {
		myApp.SetIcon(resourceFloppyIconPng)
	}

	myWindow := myApp.NewWindow("Val's Saves Binder")

	// Load saved language or fall back to system language
	prefs := myApp.Preferences()
	savedLang := prefs.StringWithFallback("app_language", GetDefaultSystemLang())
	SetLanguage(savedLang)

	binderApp := &SavesBinderApp{
		app:    myApp,
		window: myWindow,
	}

	binderApp.loadData()

	myWindow.SetContent(binderApp.buildUI())
	myWindow.Resize(fyne.NewSize(1000, 700))
	myWindow.ShowAndRun()
}
