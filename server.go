package main

import (
	// "crypto/rand"
	"database/sql"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	elastic "elastic-go"
)

const STATIC_FILE_DIR string = "/static/"
const STATIC_FILE_LOC string = "./web/static"

type Instance struct {

 	dir string

	startTime time.Time

	db *sql.DB

	es *elastic.Client

	searchIndex string

	docType string

	topics map[string]Topic

	templates map[string]*template.Template

	store *sessions.CookieStore

}

type handler struct {

	*Instance

	H func(*Instance, http.ResponseWriter, *http.Request) (int, error)

}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if status, err := h.H(h.Instance, w, r); err != nil {
		log.Println(err)
		switch status {
			case http.StatusNotFound:
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			case http.StatusUnauthorized:
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			case http.StatusInternalServerError:
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			default:
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func initInstance() (*Instance, error) {
	db, err := initDatabase(dbConfig{"postgres", "1234", "ussc_caselaw", "localhost", "5432"})
	if err != nil {
		return nil, err
	}

	es, err := elastic.New()
	if err != nil {
		return nil, err
	}

	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	key := []byte("lEMVrlJovIA7dZlTvhpFb8NSpoMtXJEaMxXrAOalJI56e0ESp7b8Ko0jrfP0A7olAK1QlOa0dlLFrT2mWK0-6w==")
	// key := make([]byte, 64)
	// _, err = rand.Read(key)
	// if err != nil {
	// 	return nil, err
	// }

	return &Instance{
		dir: dir,
		db: db,
		es: es,
		startTime: time.Now(),
		searchIndex: "ussc",
		docType: "decision",
		topics: nil,
		templates: make(map[string]*template.Template),
		store: sessions.NewCookieStore(key),
	}, nil
}

func loadConfig() error {
	// TODO ...
	return nil
}

func (i *Instance) router() *mux.Router {
	r := mux.NewRouter()

	gets := r.Methods("GET").Subrouter()
	posts := r.Methods("POST").Subrouter()

	// Views -------------------------------------------------------------------
	gets.Handle("/login", handler{i, loginViewHandler})
	posts.Handle("/lgh", handler{i, loginHandler})
	gets.Handle("/", handler{i, indexViewHandler})
	gets.Handle("/info", handler{i, infoViewHandler})

	// Decision
	gets.Handle("/decision/{docId}", handler{i, decisionViewHandler})
	gets.Handle("/ddata/{docId}", handler{i, decisionHandler})

	// Topics  -----------------------------------------------------------------
	gets.Handle("/topics", handler{i, topicIndexViewHandler})
	gets.Handle("/data", handler{i, topicIndexDataHandler})
	gets.Handle("/topic/{topicId}", handler{i, topicViewHandler})
	gets.Handle("/data/{topicId}", handler{i, topicDataHandler})
	gets.Handle("/tdata/{topicId}/{docId}", handler{i, topicDecisionHandler})

	// Database functions ------------------------------------------------------
	gets.Handle("/tags/{topicId}/{docId}", handler{i, getTagHandler})
	posts.Handle("/tag", handler{i, apiSaveTag})

	 // Searching functions ----------------------------------------------------
	posts.Handle("/search", handler{i, apiSearch})

	// Asesssments  ------------------------------------------------------------
	posts.Handle("/assess", handler{i, apiAssessTopic})

	// Add for auto annotate citations
	// gets.Handle("/citation", handler{i, apiTagCitations})

	// Serve static files
	gets.PathPrefix(STATIC_FILE_DIR).Handler(http.StripPrefix(STATIC_FILE_DIR,
		http.FileServer(http.Dir(STATIC_FILE_LOC))))

	return r
}

func main() {
	// confLocation := flag.String("c", "", "Path to config file")
	loadTopic := flag.Bool("l", false, "Load stored topics")
	topicLocation := flag.String("t", "", "Path to topic files")
	updateTopics := flag.Bool("u", false, "Update stored topics")
	flag.Parse()

	fmt.Println(
`====================================
Search classifier
(c) Daniel Locke, 2017

`)

	instance, err := initInstance()
	if err != nil {
		log.Panic(err)
	}

	topics, err := loadTopics(instance.dir, *topicLocation, *loadTopic, *updateTopics)
	if err != nil {
		log.Panic(err)
	}
	log.Println("topics loaded.")
	instance.topics = *topics
	// err = instance.getNumResultsForManualQueries()
	// if err != nil {
	// 	log.Panic(err)
	// }
	// log.Println("topics results loaded.")

	templates, err := loadTemplates(instance.dir)
	if err != nil {
		log.Panic(err)
	}
	log.Println("templates loaded.")
	instance.templates = templates

	// Load router
	r := instance.router()
	srv := &http.Server{
		Handler: r,
		Addr: ":8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Println("server started.")
	log.Fatal(srv.ListenAndServe())
}
