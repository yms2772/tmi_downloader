package main

import (
	"image/color"

	"fyne.io/fyne"
	"fyne.io/fyne/theme"
)

var (
	defFont fyne.Resource

	dark     = &color.RGBA{R: 38, G: 38, B: 40, A: 255}
	orange   = &color.RGBA{R: 198, G: 123, B: 0, A: 255}
	grey     = &color.Gray{Y: 123}
	darkGrey = &color.RGBA{R: 104, G: 104, B: 104, A: 255}
)

// customTheme is a simple demonstration of a bespoke theme loaded by a Fyne app.
type customTheme struct {
}

func (customTheme) BackgroundColor() color.Color {
	return dark
}

func (customTheme) ButtonColor() color.Color {
	return color.Black
}

func (customTheme) DisabledButtonColor() color.Color {
	return color.White
}

func (customTheme) HyperlinkColor() color.Color {
	return orange
}

func (customTheme) TextColor() color.Color {
	return color.White
}

func (customTheme) DisabledTextColor() color.Color {
	return darkGrey
}

func (customTheme) IconColor() color.Color {
	return color.White
}

func (customTheme) DisabledIconColor() color.Color {
	return color.Black
}

func (customTheme) PlaceHolderColor() color.Color {
	return grey
}

func (customTheme) PrimaryColor() color.Color {
	return darkGrey
}

func (customTheme) HoverColor() color.Color {
	return color.Black
}

func (customTheme) FocusColor() color.Color {
	return color.Black
}

func (customTheme) ScrollBarColor() color.Color {
	return grey
}

func (customTheme) ShadowColor() color.Color {
	return color.Black
}

func (customTheme) TextSize() int {
	return 12
}

func (customTheme) TextFont() fyne.Resource {
	return defFont
}

func (customTheme) TextBoldFont() fyne.Resource {
	return defFont
}

func (customTheme) TextItalicFont() fyne.Resource {
	return defFont
}

func (customTheme) TextBoldItalicFont() fyne.Resource {
	return defFont
}

func (customTheme) TextMonospaceFont() fyne.Resource {
	return theme.DefaultTextMonospaceFont()
}

func (customTheme) Padding() int {
	return 3
}

func (customTheme) IconInlineSize() int {
	return 20
}

func (customTheme) ScrollBarSize() int {
	return 5
}

func (customTheme) ScrollBarSmallSize() int {
	return 5
}

func newCustomTheme(font fyne.Resource) fyne.Theme {
	defFont = font
	return &customTheme{}
}
