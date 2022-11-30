package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/cloudprivacylabs/leap/pkg/valueset"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v2"
)

const hostName = "localhost"
const serverPort = 8011

func main() {
	cfg, err := parseYAML("testdata/cfg/queries.yaml")
	if err != nil {
		fmt.Println(err)
	}
	defer cfg.Close()
	conn, err := pgx.Connect(context.Background(), cfg.Databases[0].URI)

	if err != nil {
		log.Fatal("error connecting to db:", err)
	}
	defer conn.Close(context.Background())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", serverPort),
		Handler: route(cfg),
	}
	fmt.Printf("Server started http://%s:%d", hostName, serverPort)
	fmt.Println()
	err = server.ListenAndServe()
	// err = serveHTTP(cfg.Databases[0].Database)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Println("server closed")
	}
}

type cfgFuncHandler func(w http.ResponseWriter, r *http.Request, cfg valueset.Config)

func makeCfgFuncHandler(f cfgFuncHandler, cfg valueset.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f(w, r, cfg)
	}
}

func route(cfg valueset.Config) http.Handler {
	// router := httprouter.New()
	// router.HandlerFunc(http.MethodGet, "/", makeCfgFuncHandler(PsqlURLHandler, cfg))
	// return router
	mux := http.DefaultServeMux
	http.HandleFunc("/", makeCfgFuncHandler(PsqlURLHandler, cfg))
	return mux
}

func PsqlURLHandler(w http.ResponseWriter, r *http.Request, cfg valueset.Config) {
	// get query params from ???
	qP := r.URL.Query().Encode()
	// fmt.Println(qP)
	queryParams, err := valueset.Process("http://localhost:8011?" + qP)
	// fmt.Println(queryParams)
	if err != nil {
		fmt.Println(err)
	}
	// w.Write([]byte(fmt.Sprintf("%v", queryParams)))
	res, err := valueset.GetResults(cfg, queryParams)
	if err != nil {
		fmt.Println(err)
	}
	writeJSON(w, http.StatusOK, res, nil)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}, headers http.Header) error {
	// Encode the data to JSON, return an error if there was one
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}
	js = append(js, '\n')

	// Add any additional headers
	for key, value := range headers {
		w.Header()[key] = value
	}

	// Add the "Content-Type: application/json" header, then write
	// status code and JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

func parseYAML(filename string) (valueset.Config, error) {
	var cfg valueset.Config
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return valueset.Config{}, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return valueset.Config{}, err
	}
	loadEnv(cfg)
	return cfg, nil
}

func loadEnv(cfg valueset.Config) {
	for ix, db := range cfg.Databases {
		switch db.Adapter {
		case "pgx":
			if err := godotenv.Load("valueset/pgx.env"); err != nil {
				panic(err)
			}
			cfg.Databases[ix].Database.DatabaseName = os.Getenv("driver")
			cfg.Databases[ix].URI = os.Getenv("uri")
			return
		}
	}
}
