package renderer

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Engine struct
type Engine struct {
	Left  string
	Right string
	// views folder
	Directory string
	// views extension
	Extension string
	// layout variable name that incapsulates the template
	LayoutName string
	// determines if the engine parsed all templates
	Loaded bool
	// reload on each render
	Verbose bool
	// lock for funcmap and templates
	Mutex sync.RWMutex
	// template funcmap
	Funcmap map[string]interface{}
	// templates
	Templates *template.Template
}

func (e *Engine) AddFunc(name string, fn interface{}) *Engine {
	e.Mutex.Lock()
	e.Funcmap[name] = fn
	e.Mutex.Unlock()
	return e
}

func New(directory, extension string) *Engine {
	engine := &Engine{
		Directory:  directory,
		Extension:  extension,
		Left:       "{{",
		Right:      "}}",
		LayoutName: "embed",
		Funcmap:    make(map[string]interface{}),
	}
	engine.AddFunc(engine.LayoutName, func() error {
		return fmt.Errorf("layoutName called unexpectedly")
	})
	return engine
}

// Load parses the templates to the engine.
func (e *Engine) Load() error {
	if e.Loaded {
		return nil
	}
	// race safe
	e.Mutex.Lock()
	defer e.Mutex.Unlock()
	e.Templates = template.New(e.Directory)

	// Set template settings
	e.Templates.Delims(e.Left, e.Right)
	e.Templates.Funcs(e.Funcmap)

	walkFn := func(path string, info os.FileInfo, err error) error {
		// Return error if exist
		if err != nil {
			return err
		}
		// Skip file if it's a directory or has no file info
		if info == nil || info.IsDir() {
			return nil
		}
		// Skip file if it does not equal the given template Extension
		if len(e.Extension) >= len(path) || path[len(path)-len(e.Extension):] != e.Extension {
			return nil
		}
		// Get the relative file path
		// ./views/html/index.tmpl -> index.tmpl
		rel, err := filepath.Rel(e.Directory, path)
		if err != nil {
			return err
		}
		// Reverse slashes '\' -> '/' and
		// partials\footer.tmpl -> partials/footer.tmpl
		name := filepath.ToSlash(rel)
		// Remove ext from name 'index.tmpl' -> 'index'
		name = strings.TrimSuffix(name, e.Extension)
		// name = strings.Replace(name, e.Extension, "", -1)
		// Read the file
		// #gosec G304
		buf, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// Create new template associated with the current one
		// This enable use to invoke other templates {{ template .. }}
		_, err = e.Templates.New(name).Parse(string(buf))
		if err != nil {
			return err
		}
		// Debugging
		if e.Verbose {
			log.Printf("views: parsed template: %s\n", name)
		}
		return err
	}
	// notify engine that we parsed all templates
	e.Loaded = true
	return filepath.Walk(e.Directory, walkFn)
}

// Render will execute the template name along with the given values.
func (e *Engine) Render(out io.Writer, name string, binding interface{}, layout ...string) error {
	if !e.Loaded {
		if err := e.Load(); err != nil {
			return err
		}
	}

	tmpl := e.Templates.Lookup(name)
	if tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", name)
	}
	if len(layout) > 0 && layout[0] != "" {
		lay := e.Templates.Lookup(layout[0])
		if lay == nil {
			return fmt.Errorf("render: LayoutName %s does not exist", layout[0])
		}
		e.Mutex.Lock()
		defer e.Mutex.Unlock()
		lay.Funcs(map[string]interface{}{
			e.LayoutName: func() error {
				return tmpl.Execute(out, binding)
			},
		})
		return lay.Execute(out, binding)
	}
	return tmpl.Execute(out, binding)
}
