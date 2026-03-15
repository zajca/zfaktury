package pdf

import (
	"embed"
	"fmt"

	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/repository"
)

//go:embed fonts/*.ttf
var fontsFS embed.FS

const fontFamily = "liberation"

// loadCustomFonts reads the embedded Liberation Sans TTF files and returns
// custom font definitions for the maroto config builder.
func loadCustomFonts() ([]*entity.CustomFont, error) {
	styles := []struct {
		style fontstyle.Type
		file  string
	}{
		{fontstyle.Normal, "fonts/LiberationSans-Regular.ttf"},
		{fontstyle.Bold, "fonts/LiberationSans-Bold.ttf"},
		{fontstyle.Italic, "fonts/LiberationSans-Italic.ttf"},
		{fontstyle.BoldItalic, "fonts/LiberationSans-BoldItalic.ttf"},
	}

	repo := repository.New()
	for _, s := range styles {
		data, err := fontsFS.ReadFile(s.file)
		if err != nil {
			return nil, fmt.Errorf("reading embedded font %s: %w", s.file, err)
		}
		repo.AddUTF8FontFromBytes(fontFamily, s.style, data)
	}

	return repo.Load()
}
