package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/template"

	"github.com/rusq/osenv/v2"
	"gopkg.in/yaml.v3"
)

var base = osenv.Value("BASE_DIR", "..")

var (
	catalogue = flag.String("f", filepath.Join(base, "catalogue.yaml"), "catalogue file")
	output    = flag.String("o", filepath.Join(base, "README.md"), "output file")
	validate  = flag.Bool("v", false, "validate the catalogue file")
)

//go:embed readme.md.tmpl
var readmeTmpl string

var tmpl = template.Must(template.New("readme").Parse(readmeTmpl))

type Contribution struct {
	Title        string   `yaml:"title,omitempty"`
	Path         string   `yaml:"path,omitempty"`
	Author       string   `yaml:"author,omitempty"`
	Source       string   `yaml:"source,omitempty"`
	Description  string   `yaml:"description,omitempty"`
	Dependencies []string `yaml:"dependencies,omitempty"`
}

func main() {
	flag.Parse()

	// Read the catalogue file
	cat, err := readCatalogue(*catalogue)
	if err != nil {
		log.Fatal(err)
	}

	if *validate {
		if err := validateCatalogue(base, cat); err != nil {
			log.Fatal(err)
		}
		log.Println("Validation PASS")
	} else {
		// render
		if err := renderCatalogue(*output, tmpl, cat); err != nil {
			log.Fatal(err)
		}
		log.Println("OK")
	}
}

func readCatalogue(path string) ([]Contribution, error) {
	var contributions []Contribution
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&contributions); err != nil {
		return nil, err
	}

	return contributions, nil
}

func renderCatalogue(path string, tmpl *template.Template, contributions []Contribution) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, contributions)
}

func validateCatalogue(base string, contributions []Contribution) error {
	for _, c := range contributions {
		if c.Path == "" {
			return fmt.Errorf("missing path for contribution %q", c.Title)
		}
		_, err := os.Stat(filepath.Join(base, c.Path))
		if err != nil {
			return fmt.Errorf("invalid path for contribution %q: %w", c.Title, err)
		}
		if c.Author == "" {
			return fmt.Errorf("missing author for contribution %q", c.Title)
		}
		if c.Description == "" {
			return fmt.Errorf("missing description for contribution %q", c.Title)
		}
	}
	return nil
}
