package render

import (
	_ "embed"
	"fmt"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

// 思源黑体随程序打包，不依赖系统字体。
//
//go:embed fonts/SourceHanSansSC-Regular.otf
var regularData []byte

//go:embed fonts/SourceHanSansSC-Bold.otf
var boldData []byte

var (
	fontOnce sync.Once
	regularF *sfnt.Font
	boldF    *sfnt.Font
	fontErr  error
)

func loadFonts() error {
	fontOnce.Do(func() {
		regularF, fontErr = opentype.Parse(regularData)
		if fontErr != nil {
			fontErr = fmt.Errorf("加载思源黑体失败: %w", fontErr)
			return
		}
		boldF, fontErr = opentype.Parse(boldData)
		if fontErr != nil {
			fontErr = fmt.Errorf("加载思源黑体粗体失败: %w", fontErr)
		}
	})
	return fontErr
}

type faceKey struct {
	size int
	bold bool
}

type fontSet struct {
	faces map[faceKey]font.Face
}

func newFontSet() *fontSet {
	return &fontSet{faces: make(map[faceKey]font.Face, 16)}
}

func (f *fontSet) face(size float64, bold bool) font.Face {
	key := faceKey{size: int(size*2 + 0.5), bold: bold}
	if face, ok := f.faces[key]; ok {
		return face
	}
	src := regularF
	if bold {
		src = boldF
	}
	face, err := opentype.NewFace(src, &opentype.FaceOptions{
		Size:    float64(key.size) / 2,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	if err != nil {
		panic(err)
	}
	f.faces[key] = face
	return face
}
