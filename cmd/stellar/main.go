// Command stellar generates a CSS design-token bundle from stellar.config.json.
//
//	stellar build  [-config stellar.config.json] [-o path]   write the bundle
//	stellar init   [-config stellar.config.json]             write default config and schema
//	stellar schema                                            print the JSON Schema
//	stellar serve  [-config stellar.config.json] [-addr :7331] run the editor
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/ozzyozbourne/stellecss/internal/bundle"
	"github.com/ozzyozbourne/stellecss/internal/config"
	"github.com/ozzyozbourne/stellecss/internal/web"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "stellar:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: stellar build|init|schema|serve [flags]")
	}
	fl := flag.NewFlagSet(args[0], flag.ContinueOnError)
	cfgPath := fl.String("config", "stellar.config.json", "config file")
	switch args[0] {
	case "build":
		out := fl.String("o", "", "output path (default: bundle.output, relative to the config)")
		if err := fl.Parse(args[1:]); err != nil {
			return err
		}
		return build(*cfgPath, *out)
	case "init":
		if err := fl.Parse(args[1:]); err != nil {
			return err
		}
		return initConfig(*cfgPath)
	case "schema":
		_, err := os.Stdout.Write(config.Schema())
		return err
	case "serve":
		addr := fl.String("addr", ":7331", "listen address")
		if err := fl.Parse(args[1:]); err != nil {
			return err
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		return web.Serve(ctx, *addr, *cfgPath, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}
	return fmt.Errorf("unknown command %q", args[0])
}

func build(cfgPath, out string) error {
	c, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	b, err := bundle.Build(c)
	if err != nil {
		return err
	}
	if out == "" {
		out = filepath.Join(filepath.Dir(cfgPath), c.Bundle.Output)
	}
	if !c.Bundle.Overwrite {
		if _, err := os.Stat(out); err == nil {
			return fmt.Errorf("%s exists and bundle.overwrite is false", out)
		}
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(out, b.CSS, 0o644); err != nil {
		return err
	}
	if len(b.Map) > 0 {
		if err := os.WriteFile(out+".map", b.Map, 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("%s: %d custom properties, %s raw, %s brotli\n", out, b.Properties, bundle.KiB(b.RawSize), bundle.KiB(b.BrotliSize))
	return nil
}

func initConfig(cfgPath string) error {
	if _, err := os.Stat(cfgPath); err == nil {
		return fmt.Errorf("%s already exists", cfgPath)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := config.Default().Save(cfgPath); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(filepath.Dir(cfgPath), "stellar.config.schema.json"), config.Schema(), 0o644)
}
