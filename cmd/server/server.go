package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"
	"github.com/joho/godotenv"
	"github.com/raminderis/lenslocked/controller"
	"github.com/raminderis/lenslocked/models"
	"github.com/raminderis/lenslocked/templates"
	"github.com/raminderis/lenslocked/views"
)

type config struct {
	PSQL models.PostgresConfig
	SMTP models.SMTPConfig
	CSRF struct {
		Key    string
		Secure bool
	}
	Server struct {
		Address string
	}
}

func loadEnvConfig() (config, error) {
	var cfg config
	err := godotenv.Load()
	if err != nil {
		return cfg, err
	}
	//PSQL
	//cfg.PSQL = models.DefaultPostgresConfig()
	cfg.PSQL = models.PostgresConfig{
		Host:     os.Getenv("PSQL_HOST"),
		Port:     os.Getenv("PSQL_PORT"),
		User:     os.Getenv("PSQL_USER"),
		Password: os.Getenv("PSQL_PASSWORD"),
		DBname:   os.Getenv("PSQL_DATABASE"),
		SSLMode:  os.Getenv("PSQL_SSLMODE"),
	}
	if cfg.PSQL.Host == "" || cfg.PSQL.Port == "" {
		return cfg, fmt.Errorf("no psql config provided")
	}
	//SMTP
	cfg.SMTP.Host = os.Getenv("SMTP_HOST")
	portStr := os.Getenv("SMTP_PORT")
	cfg.SMTP.Port, err = strconv.Atoi(portStr)
	if err != nil {
		return cfg, err
	}
	cfg.SMTP.Username = os.Getenv("SMTP_USERNAME")
	cfg.SMTP.Password = os.Getenv("SMTP_PASSWORD")

	//CSRF
	cfg.CSRF.Key = os.Getenv("CSRF_KEY")
	cfg.CSRF.Secure = os.Getenv("CSRF_SECURE") == "true"
	//Server
	cfg.Server.Address = os.Getenv("SERVER_ADDRESS")
	return cfg, nil
}

func main() {
	cfg, err := loadEnvConfig()
	if err != nil {
		panic(err)
	}
	err = run(cfg)
	if err != nil {
		fmt.Println(cfg)
		panic(err)
	}
}

func run(cfg config) error {
	//Setup the DB
	//fmt.Println(cfg.PSQL)
	db, err := models.Open(cfg.PSQL)
	// cfg := models.DefaultCloudSqlConfig()
	// db, err := models.ConnectWithConnector(cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	// err = models.MigrateFS(db, migrations.FS, "")
	// if err != nil {
	// 	return err
	// }

	//Setup Services
	userService := &models.UserService{
		DB: db,
	}
	//Setup Middleware
	csrfMw := csrf.Protect([]byte(cfg.CSRF.Key), csrf.Secure(cfg.CSRF.Secure))

	//Setup controllers
	usersC := controller.Users{
		UserService: userService,
	}
	usersC.Templates.CityTemp = views.Must(views.ParseFS(
		templates.FS,
		"citytemp.gohtml", "tailwind.gohtml",
	))
	usersC.Templates.ShowCityTemp = views.Must(views.ParseFS(
		templates.FS,
		"showcitytemp.gohtml", "tailwind.gohtml",
	))
	//Setup Router and Routes
	r := chi.NewRouter()
	r.Use(csrfMw)
	r.Get("/", controller.StaticHandler(
		views.Must(views.ParseFS(templates.FS, "home.gohtml", "tailwind.gohtml"))))
	r.Get("/blogs", controller.StaticHandler(
		views.Must(views.ParseFS(templates.FS, "blog_list.gohtml", "tailwind.gohtml"))))
	r.Get("/citytemp", usersC.CityTemp)
	r.Post("/citytemp", usersC.ProcessCityTemp)
	// r.Route("/users/me", func(r chi.Router) {
	// 	r.Use(umw.RequireUser)
	// 	r.Get("/", usersC.CurrentUser)
	// })
	assetsHandler := http.FileServer(http.Dir("assets"))
	r.Get("/assets/*", http.StripPrefix("/assets", assetsHandler).ServeHTTP)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	})

	//Start the Server
	port := cfg.Server.Address
	fmt.Printf("LISTENING now on: %s...\n", port)
	return http.ListenAndServe(port, r)
}
