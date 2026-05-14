package cards

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

type ArtRenderer struct {
	CacheDir   string
	HTTPClient *http.Client
}

func (r ArtRenderer) Render(ctx context.Context, imageURL string, width, height int) (string, error) {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return "No image available for this card.", nil
	}
	if width < 8 || height < 4 {
		return "Resize the terminal to show card art.", nil
	}
	localPath, err := r.cachedImage(ctx, imageURL)
	if err != nil {
		return "", err
	}
	f, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return "", err
	}
	art := renderHalfBlockImage(img, width, height)
	if strings.TrimSpace(stripANSILite(art)) == "" {
		return "Image converted to an empty preview.", nil
	}
	return art, nil
}

func renderHalfBlockImage(img image.Image, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, height*2))
	draw.CatmullRom.Scale(dst, fitRect(dst.Bounds(), img.Bounds()), img, img.Bounds(), draw.Over, nil)
	var b strings.Builder
	for y := 0; y < height; y++ {
		if y > 0 {
			b.WriteByte('\n')
		}
		for x := 0; x < width; x++ {
			fg := rgba8(dst.At(x, y*2))
			bg := rgba8(dst.At(x, min(y*2+1, height*2-1)))
			switch {
			case fg.A == 0 && bg.A == 0:
				b.WriteString("\x1b[0m ")
			case fg.A == 0:
				fmt.Fprintf(&b, "\x1b[0m\x1b[38;2;%d;%d;%dm▄", bg.R, bg.G, bg.B)
			case bg.A == 0:
				fmt.Fprintf(&b, "\x1b[0m\x1b[38;2;%d;%d;%dm▀", fg.R, fg.G, fg.B)
			default:
				fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀", fg.R, fg.G, fg.B, bg.R, bg.G, bg.B)
			}
		}
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

func fitRect(dst, src image.Rectangle) image.Rectangle {
	dstW, dstH := dst.Dx(), dst.Dy()
	srcW, srcH := src.Dx(), src.Dy()
	if dstW <= 0 || dstH <= 0 || srcW <= 0 || srcH <= 0 {
		return dst
	}
	targetW := dstW
	targetH := int(float64(targetW) * float64(srcH) / float64(srcW))
	if targetH > dstH {
		targetH = dstH
		targetW = int(float64(targetH) * float64(srcW) / float64(srcH))
	}
	targetW = max(1, targetW)
	targetH = max(1, targetH)
	x0 := dst.Min.X + (dstW-targetW)/2
	y0 := dst.Min.Y + (dstH-targetH)/2
	return image.Rect(x0, y0, x0+targetW, y0+targetH)
}

func rgba8(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	if a == 0 {
		return color.RGBA{}
	}
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

func (r ArtRenderer) cachedImage(ctx context.Context, imageURL string) (string, error) {
	if strings.TrimSpace(r.CacheDir) == "" {
		return "", fmt.Errorf("image cache dir is required")
	}
	if err := os.MkdirAll(r.CacheDir, 0o755); err != nil {
		return "", err
	}
	ext := ".img"
	if parsed, err := url.Parse(imageURL); err == nil {
		if pathExt := strings.TrimSpace(path.Ext(parsed.Path)); pathExt != "" && len(pathExt) <= 8 {
			ext = pathExt
		}
	}
	sum := sha256.Sum256([]byte(imageURL))
	localPath := filepath.Join(r.CacheDir, hex.EncodeToString(sum[:])+ext)
	if info, err := os.Stat(localPath); err == nil && !info.IsDir() && info.Size() > 0 {
		return localPath, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := r.httpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download card image: HTTP %d", resp.StatusCode)
	}
	tmp, err := os.CreateTemp(r.CacheDir, ".image.*.tmp")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if _, err := io.Copy(tmp, io.LimitReader(resp.Body, 20<<20)); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmpPath, localPath); err != nil {
		return "", err
	}
	return localPath, nil
}

func (r ArtRenderer) httpClient() *http.Client {
	if r.HTTPClient != nil {
		return r.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func stripANSILite(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if inEsc {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEsc = false
			}
			continue
		}
		if r == '\x1b' {
			inEsc = true
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
