package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"micro-front/internal/blogs"
	"micro-front/internal/config"
	"micro-front/internal/images"
	"micro-front/internal/publish"
	"micro-front/internal/seed"
	"micro-front/internal/server"
	"micro-front/internal/site"
	"micro-front/internal/store"
	"micro-front/internal/titleimage"
	"micro-front/internal/web"
)

func run(ctx context.Context, args []string) error {
	cfg := config.Load()

	if len(args) > 0 && args[0] == "publish" {
		return runPublish(ctx, cfg, args[1:])
	}
	if len(args) > 0 && args[0] == "seed" {
		return runSeed(ctx, cfg, args[1:])
	}
	if len(args) > 0 && args[0] == "sample" {
		return runSample(ctx, args[1:])
	}

	st, err := store.New(cfg.DataDir)
	if err != nil {
		return err
	}
	defer st.Close()

	srv := server.New(cfg)

	site.Handler{Store: st}.Init(&srv)
	blogs.Handler{Store: st, DataDir: cfg.DataDir}.Init(&srv)
	images.Handler{Store: st, DataDir: cfg.DataDir}.Init(&srv)
	titleimage.Handler{}.Init(&srv)
	web.Handler{Store: st, DataDir: cfg.DataDir, PublishDir: cfg.PublicStaticDir}.Init(&srv)

	return srv.Run(ctx)
}

func runSeed(ctx context.Context, cfg config.Config, args []string) error {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	fs.SetOutput(log.Writer())
	profile := fs.String("profile", "default", "seed profile under ./seeds")
	seedDir := fs.String("seed-dir", "", "seed data directory")
	reset := fs.Bool("reset", true, "reset blogs and images before seeding")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *seedDir == "" {
		*seedDir = "seeds/" + *profile
	}

	st, err := store.New(cfg.DataDir)
	if err != nil {
		return err
	}
	defer st.Close()

	if err := (seed.Usecase{Store: st, DataDir: cfg.DataDir}).Run(ctx, seed.Options{
		SeedDir: *seedDir,
		Reset:   *reset,
	}); err != nil {
		return err
	}

	log.Printf("seed complete: seed_dir=%s reset=%v", *seedDir, *reset)
	return nil
}

func runPublish(ctx context.Context, cfg config.Config, args []string) error {
	fs := flag.NewFlagSet("publish", flag.ContinueOnError)
	fs.SetOutput(log.Writer())
	target := fs.String("target", "all", "publish target")
	blogID := fs.Int64("blog-id", 0, "blog id for blog target")
	outputDir := fs.String("publish-dir", cfg.PublicStaticDir, "publish output dir")
	if err := fs.Parse(args); err != nil {
		return err
	}

	st, err := store.New(cfg.DataDir)
	if err != nil {
		return err
	}
	defer st.Close()

	uc := publish.Usecase{
		Store:      st,
		PublishDir: *outputDir,
	}
	_, fields, err := uc.Run(ctx, publish.Request{
		Target: *target,
		BlogID: *blogID,
	})
	if len(fields) > 0 {
		return fmt.Errorf("validation failed: target=%v blog_id=%v", fields["target"], fields["blog_id"])
	}
	if err != nil {
		return err
	}

	log.Printf("publish complete: target=%s blog_id=%d output_dir=%s", *target, *blogID, *outputDir)
	return nil
}

func runSample(_ context.Context, args []string) error {
	fs := flag.NewFlagSet("sample", flag.ContinueOnError)
	fs.SetOutput(log.Writer())
	outputDir := fs.String("output-dir", "docs/mocks/blog-header", "sample SVG output directory")
	title := fs.String("title", "ブログタイトル画像ジェネレータ", "sample title")
	category := fs.String("category", "Tech", "sample category")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		return fmt.Errorf("create sample output dir: %w", err)
	}
	for _, tmpl := range titleimage.ListTemplates() {
		svg, err := titleimage.GenerateSVG(titleimage.GenerateInput{
			Title:    *title,
			Category: *category,
			Template: tmpl.ID,
		})
		if err != nil {
			return err
		}
		path := filepath.Join(*outputDir, string(tmpl.ID)+".svg")
		if err := os.WriteFile(path, []byte(svg), 0o644); err != nil {
			return fmt.Errorf("write sample svg %s: %w", path, err)
		}
		log.Printf("sample generated: %s", path)
	}

	return nil
}
