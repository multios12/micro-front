package images

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"micro-front/internal/store"
)

func TestHandlerPostImagesOrientation(t *testing.T) {
	// Each source corner has a distinct grayscale value, clockwise from top left.
	src := image.NewGray(image.Rect(0, 0, 80, 48))
	for y := 0; y < 48; y++ {
		for x := 0; x < 80; x++ {
			value := uint8(30)
			if x >= 40 && y < 24 {
				value = 90
			}
			if x >= 40 && y >= 24 {
				value = 150
			}
			if x < 40 && y >= 24 {
				value = 210
			}
			src.SetGray(x, y, color.Gray{Y: value})
		}
	}
	var original bytes.Buffer
	if err := jpeg.Encode(&original, src, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		orientation uint16
		corners     [4]uint8
	}{
		{0, [4]uint8{30, 90, 150, 210}}, // No EXIF.
		{1, [4]uint8{30, 90, 150, 210}},
		{2, [4]uint8{90, 30, 210, 150}},
		{3, [4]uint8{150, 210, 30, 90}},
		{4, [4]uint8{210, 150, 90, 30}},
		{5, [4]uint8{30, 210, 150, 90}},
		{6, [4]uint8{210, 30, 90, 150}},
		{7, [4]uint8{150, 90, 30, 210}},
		{8, [4]uint8{90, 150, 210, 30}},
		{9, [4]uint8{30, 90, 150, 210}}, // Invalid orientation leaves pixels unchanged.
	}
	for _, tc := range cases {
		t.Run(fmt.Sprint(tc.orientation), func(t *testing.T) {
			dir := t.TempDir()
			st, err := store.New(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			blog, err := st.CreateBlog(context.Background(), store.BlogEntitty{Title: "orientation", Status: "private"})
			if err != nil {
				t.Fatal(err)
			}
			input := original.Bytes()
			if tc.orientation != 0 {
				// APP1 contains a little-endian TIFF IFD with the Orientation SHORT tag.
				exif := []byte{'E', 'x', 'i', 'f', 0, 0, 'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 1, 3, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
				binary.LittleEndian.PutUint16(exif[24:26], tc.orientation)
				segment := []byte{0xff, 0xe1, 0, byte(len(exif) + 2)}
				segment = append(segment, exif...)
				input = append(append(append([]byte{}, input[:2]...), segment...), input[2:]...)
			}
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("file", "photo.jpg")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := part.Write(input); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, "/", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.SetPathValue("blog_id", strconv.FormatInt(blog.ID, 10))
			rec := httptest.NewRecorder()
			Handler{Store: st, DataDir: dir}.handlerPostImages(rec, req)
			if rec.Code != http.StatusCreated {
				t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
			}
			var resp ImagesUploadResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatal(err)
			}
			saved, err := os.ReadFile(filepath.Join(dir, strings.TrimPrefix(resp.URL, "/admin/")))
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(saved, []byte("Exif\x00\x00")) {
				t.Fatal("EXIF remains in saved JPEG")
			}
			dst, err := jpeg.Decode(bytes.NewReader(saved))
			if err != nil {
				t.Fatal(err)
			}
			w, h := 80, 48
			if tc.orientation >= 5 && tc.orientation <= 8 {
				w, h = h, w
			}
			if dst.Bounds().Dx() != w || dst.Bounds().Dy() != h {
				t.Fatalf("bounds = %v, want %dx%d", dst.Bounds(), w, h)
			}
			points := []image.Point{{8, 8}, {w - 9, 8}, {w - 9, h - 9}, {8, h - 9}}
			for i, p := range points {
				got := color.GrayModel.Convert(dst.At(p.X, p.Y)).(color.Gray).Y
				delta := int(got) - int(tc.corners[i])
				if delta < -5 || delta > 5 {
					t.Errorf("corner %d = %d, want %d", i, got, tc.corners[i])
				}
			}
		})
	}
}
