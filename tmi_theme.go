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

//customTheme is a simple demonstration of a bespoke theme loaded by a Fyne app.
type customTheme struct {
}

//BackgroundColor 백그라운드 색
func (customTheme) BackgroundColor() color.Color {
	return dark
}

//ButtonColor 버튼 색
func (customTheme) ButtonColor() color.Color {
	return color.Black
}

//DisabledButtonColor 비활성화 버튼 색
func (customTheme) DisabledButtonColor() color.Color {
	return color.White
}

//HyperlinkColor 하이퍼링크 색
func (customTheme) HyperlinkColor() color.Color {
	return orange
}

//TextColor 글자 색
func (customTheme) TextColor() color.Color {
	return color.White
}

//DisabledTextColor 비활성화 글자 색
func (customTheme) DisabledTextColor() color.Color {
	return darkGrey
}

//IconColor 아이콘 색
func (customTheme) IconColor() color.Color {
	return color.White
}

//DisabledIconColor 비활성화 아이콘 색
func (customTheme) DisabledIconColor() color.Color {
	return color.Black
}

//PlaceHolderColor 안내 문구 색
func (customTheme) PlaceHolderColor() color.Color {
	return grey
}

//PrimaryColor 기본 색
func (customTheme) PrimaryColor() color.Color {
	return darkGrey
}

//HoverColor 호버 색
func (customTheme) HoverColor() color.Color {
	return color.Black
}

//FocusColor 강조 색
func (customTheme) FocusColor() color.Color {
	return color.Black
}

//ScrollBarColor 스크롤 바 색
func (customTheme) ScrollBarColor() color.Color {
	return grey
}

//ShadowColor 그림자 색
func (customTheme) ShadowColor() color.Color {
	return color.Black
}

//TextSize 글씨 색
func (customTheme) TextSize() int {
	return 12
}

//TextFont 폰트
func (customTheme) TextFont() fyne.Resource {
	return defFont
}

//TextBoldFont 볼드 폰트
func (customTheme) TextBoldFont() fyne.Resource {
	return defFont
}

//TextItalicFont 이탈릭 폰트
func (customTheme) TextItalicFont() fyne.Resource {
	return defFont
}

//TextBoldItalicFont 볼드 이탈릭 폰트
func (customTheme) TextBoldItalicFont() fyne.Resource {
	return defFont
}

//TextMonospaceFont 모노스페이스 폰트
func (customTheme) TextMonospaceFont() fyne.Resource {
	return theme.DefaultTextMonospaceFont()
}

//Padding 패딩
func (customTheme) Padding() int {
	return 3
}

//IconInlineSize 아이콘 인라인 크기
func (customTheme) IconInlineSize() int {
	return 20
}

//ScrollBarSize 스크롤 바 크기
func (customTheme) ScrollBarSize() int {
	return 5
}

//ScrollBarSmallSize 작은 스크롤 바 크기
func (customTheme) ScrollBarSmallSize() int {
	return 5
}

// NewCustomTheme 커스텀 테마
func NewCustomTheme(font fyne.Resource) fyne.Theme {
	defFont = font
	return &customTheme{}
}
