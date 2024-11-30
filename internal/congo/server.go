package congo

import (
	"cmp"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
)

type Server struct {
	Controllers map[string]Controller
	Database    *Database
	Templates   *template.Template
	serveMux    *http.ServeMux
}

func NewServer(opts ...ServerOpt) *Server {
	server := Server{
		Controllers: map[string]Controller{},
		serveMux:    http.NewServeMux(),
	}
	for _, opt := range opts {
		Must(opt(&server))
	}
	return &server
}

func (server *Server) Serve(name string) (view View) {
	view.Server = server
	if view.template = server.Templates.Lookup(name); view.template == nil {
		log.Fatalf("Template %s not found", name)
	}
	return view
}

func (server Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	server.serveMux.ServeHTTP(w, r)
}

type ServerOpt func(*Server) error

func WithController(name string, ctrl Controller) ServerOpt {
	return func(server *Server) error {
		return server.WithController(name, ctrl)
	}
}

func (server *Server) WithController(name string, controller Controller) error {
	server.Controllers[name] = controller
	return controller.Mount(server)
}

func WithDatabase(db *Database) ServerOpt {
	return func(server *Server) error {
		return server.WithDatabase(db)
	}
}

func (server *Server) WithDatabase(db *Database) error {
	log.Println("Loading data from", db.Root)
	server.Database = db
	return db.MigrateUp()
}

func WithTemplates(templates fs.FS) ServerOpt {
	return func(server *Server) error {
		return server.WithTemplates(templates)
	}
}

func (server *Server) WithTemplates(templates fs.FS, patterns ...string) error {
	funcs := template.FuncMap{
		"db": func() *Database { return server.Database },
		"host": func() string {
			if env := os.Getenv("HOME"); env != "/home/coder" {
				return ""
			}
			port := cmp.Or(os.Getenv("PORT"), "5000")
			return fmt.Sprintf("/workspace-cgk/proxy/%s", port)
		},
	}
	for name, ctrl := range server.Controllers {
		funcs[name] = func() Controller { return ctrl }
	}
	server.Templates = template.New("").Funcs(funcs)
	if tmpl, err := server.Templates.ParseFS(templates, "templates/*.html"); err == nil {
		server.Templates = tmpl
	}
	if tmpl, err := server.Templates.ParseFS(templates, "templates/**/*.html"); err == nil {
		server.Templates = tmpl
	}

	return nil
}

func WithEndpoint(path string, api Endpoint) ServerOpt {
	return func(server *Server) error {
		return server.WithEndpoint(path, api)
	}
}

func (server *Server) WithEndpoint(path string, api Endpoint) error {
	server.serveMux.Handle(path, api)
	return nil
}
