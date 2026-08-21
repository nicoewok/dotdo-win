# dotdo-win: Fyne GUI & MonochromeTheme Reference

This document preserves the Fyne GUI implementation and `MonochromeTheme` configuration for `dotdo-win`. Re-enable this when using a CGO environment supported by your toolchain (such as MSYS2 MinGW-w64).

## 1. Monochrome Theme Implementation (`pkg/ui/theme.go`)

```go
package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type MonochromeTheme struct{}

var _ fyne.Theme = (*MonochromeTheme)(nil)

func (m *MonochromeTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.RGBA{R: 14, G: 14, B: 14, A: 255} // #0E0E0E
	case theme.ColorNameHeaderBackground, theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground:
		return color.RGBA{R: 22, G: 22, B: 22, A: 255} // #161616
	case theme.ColorNameButton:
		return color.RGBA{R: 30, G: 30, B: 30, A: 255} // #1E1E1E
	case theme.ColorNameHover:
		return color.RGBA{R: 45, G: 45, B: 45, A: 255}
	case theme.ColorNameForeground:
		return color.RGBA{R: 235, G: 235, B: 235, A: 255} // #EBEBEB
	case theme.ColorNamePrimary:
		return color.RGBA{R: 200, G: 200, B: 200, A: 255} // #C8C8C8
	case theme.ColorNameInputBackground:
		return color.RGBA{R: 24, G: 24, B: 24, A: 255}
	case theme.ColorNameShadow:
		return color.RGBA{R: 0, G: 0, B: 0, A: 160}
	case theme.ColorNamePlaceHolder:
		return color.RGBA{R: 110, G: 110, B: 110, A: 255}
	case theme.ColorNameDisabled:
		return color.RGBA{R: 65, G: 65, B: 65, A: 255}
	case theme.ColorNameSeparator:
		return color.RGBA{R: 40, G: 40, B: 40, A: 255}
	case theme.ColorNameScrollBar:
		return color.RGBA{R: 55, G: 55, B: 55, A: 255}
	case theme.ColorNameFocus:
		return color.RGBA{R: 160, G: 160, B: 160, A: 255}
	case theme.ColorNameError:
		return color.RGBA{R: 235, G: 87, B: 87, A: 255}
	case theme.ColorNameSelection:
		return color.RGBA{R: 50, G: 50, B: 50, A: 255}
	default:
		return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
}

func (m *MonochromeTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m *MonochromeTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m *MonochromeTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 13
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInnerPadding:
		return 6
	default:
		return theme.DefaultTheme().Size(name)
	}
}
```

## 2. Main Entry Point for GUI (`main.go`)

```go
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/nicoewok/dotdo-win/pkg/service"
	"github.com/nicoewok/dotdo-win/pkg/ui"
)

func main() {
	myApp := app.NewWithID("com.dotdo.win")
	myApp.Settings().SetTheme(&ui.MonochromeTheme{})

	win := myApp.NewWindow("dotdo")
	win.Resize(fyne.NewSize(540, 650))

	svc := service.NewService()
	_ = svc.Init()

	ui.SetupGUI(win, svc)

	win.ShowAndRun()
}
```

