package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	myApp := app.NewWithID("com.dotdo.win")
	myApp.Settings().SetTheme(&MonochromeTheme{})

	win := myApp.NewWindow("dotdo")
	win.Resize(fyne.NewSize(540, 650))

	svc := NewService()
	_ = svc.Init()

	SetupGUI(win, svc)

	win.ShowAndRun()
}
