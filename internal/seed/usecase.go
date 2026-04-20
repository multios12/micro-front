package seed

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"micro-front/internal/store"
	"micro-front/internal/validate"
)

type Usecase struct {
	Store   *store.Store
	DataDir string
}

func (uc Usecase) Run(ctx context.Context, opts Options) error {
	if opts.SeedDir == "" {
		opts.SeedDir = filepath.Join("seeds", "default")
	}

	if opts.Reset {
		if err := uc.reset(ctx); err != nil {
			return err
		}
	}
	if err := uc.seedSite(ctx, opts.SeedDir); err != nil {
		return err
	}
	if err := uc.seedBlogs(ctx, opts.SeedDir); err != nil {
		return err
	}
	return uc.seedImages(ctx, opts.SeedDir)
}

func (uc Usecase) reset(ctx context.Context) error {
	if _, err := uc.Store.DB.ExecContext(ctx, `DELETE FROM images`); err != nil {
		return err
	}
	if _, err := uc.Store.DB.ExecContext(ctx, `DELETE FROM blogs`); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(uc.DataDir, "images"))
}

func (uc Usecase) seedSite(ctx context.Context, seedDir string) error {
	siteSeed, err := loadJSON[SiteSeed](filepath.Join(seedDir, "site.json"))
	if err != nil {
		return err
	}
	_, err = uc.Store.UpdateSiteSettings(ctx, store.SiteEntitty{
		SiteTitle:       siteSeed.SiteTitle,
		SiteSubtitle:    siteSeed.SiteSubtitle,
		SiteDescription: siteSeed.SiteDescription,
		Tabs:            siteSeed.Tabs,
		FootInformation: siteSeed.FootInformation,
		Copyright:       siteSeed.Copyright,
	})
	return err
}

func (uc Usecase) seedBlogs(ctx context.Context, seedDir string) error {
	blogs, err := loadJSON[[]BlogSeed](filepath.Join(seedDir, "blogs.json"))
	if err != nil {
		return err
	}
	for _, item := range blogs {
		content := item.Content
		switch {
		case item.UseSampleContent:
			body, err := os.ReadFile(filepath.Join("docs", "sample-md.md"))
			if err != nil {
				return err
			}
			content = string(body)
		case item.ContentFile != "":
			body, err := readSeedText(seedDir, item.ContentFile)
			if err != nil {
				return err
			}
			content = body
		}

		blog := store.BlogEntitty{
			Title:       item.Title,
			Content:     content,
			Summary:     validate.SummaryFromContent(content),
			Category:    item.Category,
			Status:      item.Status,
			PublishedAt: item.PublishedAt,
		}
		if item.ID > 0 {
			if _, err := uc.Store.CreateBlogWithID(ctx, blog, item.ID); err != nil {
				return fmt.Errorf("seed blog id=%d title=%s: %w", item.ID, item.Title, err)
			}
			continue
		}
		if _, err := uc.Store.CreateBlog(ctx, blog); err != nil {
			return fmt.Errorf("seed blog title=%s: %w", item.Title, err)
		}
	}
	return nil
}

func (uc Usecase) seedImages(ctx context.Context, seedDir string) error {
	images, err := loadJSON[[]ImageSeed](filepath.Join(seedDir, "images.json"))
	if err != nil {
		return err
	}
	for _, item := range images {
		if item.ID <= 0 {
			return fmt.Errorf("image id is required for blog_id=%d file=%s", item.BlogID, item.File)
		}
		if _, err := uc.Store.CreateImageWithID(ctx, item.BlogID, item.ID, item.AltText); err != nil {
			return fmt.Errorf("seed image id=%d blog_id=%d: %w", item.ID, item.BlogID, err)
		}
		if err := copyImage(seedDir, uc.DataDir, item); err != nil {
			return err
		}
	}
	return nil
}

func copyImage(seedDir, dataDir string, item ImageSeed) error {
	srcPath := filepath.Join(seedDir, item.File)
	in, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer in.Close()

	dstPath := filepath.Join(dataDir, "images", fmt.Sprintf("%d", item.BlogID), fmt.Sprintf("%d.png", item.ID))
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if !strings.HasSuffix(strings.ToLower(dstPath), ".png") {
		return fmt.Errorf("seed image destination must be png: %s", dstPath)
	}
	return nil
}
