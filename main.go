package main

import (
	"database/sql"
	"flag"
	"fmt"
	"hex-example/database/psql"
	redisdb "hex-example/database/redis"
	"hex-example/ticket"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi"
	"github.com/go-redis/redis"
	_ "github.com/lib/pq"
)

func main() {

	dbType := flag.String("database", "redis", "database type [redis, psql]")
	flag.Parse()

	var ticketRepo ticket.TicketRepository

	switch *dbType {
	case "psql":
		ticketRepo = psql.NewPostgresTicketRepository(postgresConnection("postgresql://postgres@localhost/ticket?sslmode=disable"))
	case "redis":
		ticketRepo = redisdb.NewRedisTicketRepository(redisConnection("localhost:6379"))
	default:
		panic("Unknown database")
	}

	ticketService := ticket.NewTicketService(ticketRepo)
	ticketHandler := ticket.NewTicketHandler(ticketService)

	router := chi.NewRouter()
	router.Get("/tickets", ticketHandler.Get)
	router.Get("/tickets/{id}", ticketHandler.GetById)
	router.Post("/tickets", ticketHandler.Create)

	http.Handle("/", accessControl(router))

	errs := make(chan error, 2)
	go func() {
		fmt.Println("Listening on port :3000")
		errs <- http.ListenAndServe(":3000", nil)
	}()
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	fmt.Printf("terminated %s", <-errs)

}

func redisConnection(url string) *redis.Client {
	fmt.Println("Connecting to Redis DB")
	client := redis.NewClient(&redis.Options{
		Addr:     url,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	err := client.Ping().Err()

	if err != nil {
		panic(err)
	}
	return client
}

func postgresConnection(database string) *sql.DB {
	fmt.Println("Connecting to PostgreSQL DB")
	db, err := sql.Open("postgres", database)
	if err != nil {
		log.Fatalf("%s", err)
		panic(err)
	}
	return db
}

func accessControl(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type")

		if r.Method == "OPTIONS" {
			return
		}

		h.ServeHTTP(w, r)
	})
}
