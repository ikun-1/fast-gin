package main

import (
	"flag"
	"fmt"
	"os"

	"fast-gin/config"
	"fast-gin/models"

	"go.yaml.in/yaml/v3"
	"gorm.io/gen"
	"gorm.io/gorm"
)

func main() {
	configPath := flag.String("f", "settings-dev.yaml", "config file path")
	outPath := flag.String("out", "dal/query", "gorm gen output path")
	flag.Parse()

	cfg, err := readConfig(*configPath)
	if err != nil {
		panic(err)
	}

	dialector := cfg.DB.Dsn()
	if dialector == nil {
		panic("database dialector is nil")
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		panic(err)
	}

	g := gen.NewGenerator(gen.Config{
		OutPath: *outPath,
		Mode:    gen.WithDefaultQuery | gen.WithQueryInterface,
	})
	g.UseDB(db)

	g.ApplyBasic(
		models.User{},
		models.Role{},
		models.Permission{},
		models.UserRole{},
		models.RolePermission{},
		models.Image{},
	)
	g.Execute()

	fmt.Printf("gorm gen finished, output: %s\n", *outPath)
}

func readConfig(path string) (*config.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := new(config.Config)
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
