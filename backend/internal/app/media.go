package app

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/disintegration/imaging"
	"github.com/go-chi/chi/v5"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const maxPhotoBytes = 8 * 1024 * 1024

var mediaFilename = regexp.MustCompile(`^[a-f0-9]{32}\.jpg$`)

func (a *App) uploadMedia(w http.ResponseWriter, r *http.Request) error {
	if !a.allow(r, "upload", 12) {
		return &problem{Status: 429, Code: "RATE_LIMITED", Message: "Unggahan terlalu sering. Coba lagi dalam satu menit."}
	}
	if a.uploadDirectory == "" {
		return &problem{Status: 503, Code: "UPLOAD_UNAVAILABLE", Message: "Penyimpanan foto belum disiapkan."}
	}
	select {
	case a.uploadSlots <- struct{}{}:
		defer func() { <-a.uploadSlots }()
	default:
		return &problem{Status: 503, Code: "UPLOAD_BUSY", Message: "Pemrosesan foto sedang penuh. Coba unggah lagi sebentar lagi."}
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxPhotoBytes+1024*1024)
	multipart, err := r.MultipartReader()
	if err != nil {
		return invalid("Kirim satu foto melalui formulir unggahan.")
	}
	part, err := multipart.NextPart()
	if err != nil || part.FormName() != "file" || part.FileName() == "" {
		return invalid("Pilih satu foto JPG, PNG, atau WebP.")
	}
	contents, err := io.ReadAll(io.LimitReader(part, maxPhotoBytes+1))
	part.Close()
	if err != nil {
		return invalid("Foto belum dapat dibaca.")
	}
	if len(contents) > maxPhotoBytes {
		return invalid("Ukuran foto maksimal 8 MB.")
	}
	if _, err = multipart.NextPart(); err != io.EOF {
		return invalid("Unggah satu foto setiap permintaan.")
	}
	contentType := http.DetectContentType(contents)
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		return invalid("Pilih foto JPG, PNG, atau WebP yang valid.")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(contents))
	if err != nil || (format != "jpeg" && format != "png" && format != "webp") || config.Width < 1 || config.Height < 1 || config.Width > 8000 || config.Height > 8000 || int64(config.Width)*int64(config.Height) > 24_000_000 {
		return invalid("Foto harus JPG, PNG, atau WebP yang valid, maksimal 24 megapiksel.")
	}
	decoded, err := imaging.Decode(bytes.NewReader(contents), imaging.AutoOrientation(true))
	if err != nil {
		return invalid("File gambar rusak atau formatnya tidak didukung.")
	}
	config.Width, config.Height = decoded.Bounds().Dx(), decoded.Bounds().Dy()
	scale := math.Min(1, 2200/float64(max(config.Width, config.Height)))
	width, height := max(1, int(math.Round(float64(config.Width)*scale))), max(1, int(math.Round(float64(config.Height)*scale)))
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	xdraw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.White}, image.Point{}, xdraw.Src)
	xdraw.CatmullRom.Scale(canvas, canvas.Bounds(), decoded, decoded.Bounds(), xdraw.Over, nil)
	var encoded bytes.Buffer
	if err = jpeg.Encode(&encoded, canvas, &jpeg.Options{Quality: 88}); err != nil {
		return err
	}
	filename := randomHex(16) + ".jpg"
	if err = os.MkdirAll(a.uploadDirectory, 0750); err != nil {
		return fmt.Errorf("photo storage unavailable: %w", err)
	}
	destination := filepath.Join(a.uploadDirectory, filename)
	if err = os.WriteFile(destination, encoded.Bytes(), 0640); err != nil {
		return err
	}
	asset := MediaAsset{ID: newID(), URL: "/media/" + filename, Width: width, Height: height, Size: encoded.Len()}
	_, err = a.db.Exec(r.Context(), "INSERT INTO media_assets(id,filename,width,height,size_bytes,uploaded_by) VALUES($1,$2,$3,$4,$5,$6)", asset.ID, filename, width, height, asset.Size, identity(r).User.ID)
	if err != nil {
		os.Remove(destination)
		return err
	}
	return respond(w, 201, asset)
}

func (a *App) media(w http.ResponseWriter, r *http.Request) error {
	filename := chi.URLParam(r, "name")
	if !mediaFilename.MatchString(filename) || a.uploadDirectory == "" {
		return &problem{Status: 404, Code: "NOT_FOUND", Message: "Foto tidak tersedia."}
	}
	var published bool
	err := a.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM media_assets m WHERE m.filename=$1 AND (
        EXISTS(SELECT 1 FROM shop WHERE hero_image_url=$2 OR logo_image_url=$2) OR
        EXISTS(SELECT 1 FROM gallery_photos WHERE image_url=$2 AND active AND consent_confirmed)))`, filename, "/media/"+filename).Scan(&published)
	if err != nil {
		return err
	}
	if !published {
		auth, valid, err := a.sessionFromRequest(r)
		if err != nil {
			return err
		}
		if !valid || auth.User.Role != "admin" {
			return &problem{Status: 404, Code: "NOT_FOUND", Message: "Foto tidak tersedia."}
		}
	}
	file, err := os.Open(filepath.Join(a.uploadDirectory, filename))
	if os.IsNotExist(err) {
		return &problem{Status: 404, Code: "NOT_FOUND", Message: "Foto tidak tersedia."}
	}
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, filename, info.ModTime(), file)
	return nil
}
