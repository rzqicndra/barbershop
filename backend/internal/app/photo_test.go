package app

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/jpeg"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrationPhonePhotoOrientation(t *testing.T) {
	f := newFixture(t)
	admin := f.login(t, "owner@gilir.test", "Owner-password-2026")
	f.app.uploadDirectory = t.TempDir()
	var source bytes.Buffer
	if err := jpeg.Encode(&source, image.NewRGBA(image.Rect(0, 0, 20, 12)), nil); err != nil {
		t.Fatal(err)
	}
	// Minimal little-endian EXIF tag: Orientation=6 (90 degrees clockwise).
	exif := []byte{'E', 'x', 'i', 'f', 0, 0, 'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 0x01, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0}
	header := []byte{0xff, 0xe1, 0, 0}
	binary.BigEndian.PutUint16(header[2:], uint16(len(exif)+2))
	photo := append(append(append(append([]byte{}, source.Bytes()[:2]...), header...), exif...), source.Bytes()[2:]...)
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, _ := form.CreateFormFile("file", "phone.jpg")
	part.Write(photo)
	form.Close()
	req := httptest.NewRequest("POST", "/api/v1/admin/media", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("Origin", "http://gilir.test")
	req.Header.Set("X-CSRF-Token", admin.csrf)
	req.AddCookie(admin.cookie)
	response := httptest.NewRecorder()
	f.handler.ServeHTTP(response, req)
	expectStatus(t, response, 201)
	var asset MediaAsset
	if err := json.Unmarshal(response.Body.Bytes(), &asset); err != nil {
		t.Fatal(err)
	}
	if asset.Width != 12 || asset.Height != 20 {
		t.Fatalf("phone orientation lost: %dx%d", asset.Width, asset.Height)
	}
	contents, err := os.ReadFile(filepath.Join(f.app.uploadDirectory, strings.TrimPrefix(asset.URL, "/media/")))
	if err != nil {
		t.Fatal(err)
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(contents))
	if err != nil || config.Width != 12 || config.Height != 20 {
		t.Fatalf("stored rotation incorrect: %+v %v", config, err)
	}
	if bytes.Contains(contents, []byte("Exif")) {
		t.Fatal("photo metadata was retained")
	}
}
