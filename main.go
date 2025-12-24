package main

import _ "github.com/lib/pq"
import (
	"github.com/Baehry/gator/internal/config"
	"github.com/Baehry/gator/internal/database"
	"database/sql"
	"os"
	"fmt"
)

func main() {
	cfg, err := config.Read()
	db, err := sql.Open("postgres", cfg.Db_url)
	if err != nil {
		fmt.Printf("%v.\n", err)
		os.Exit(1)
	}
	dbQueries := database.New(db)
	s := state{
		db: dbQueries,
		cfgPtr: &cfg,
	}
	c := commands{
		commands: make(map[string]func(*state, command) error),
	}
	c.register("login", handlerLogin)
	c.register("register", handlerRegister)
	c.register("reset", handlerReset)
	c.register("users", handlerUsers)
	c.register("agg", handlerAgg)
	c.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	c.register("feeds", handlerFeeds)
	c.register("follow", middlewareLoggedIn(handlerFollow))
	c.register("following", middlewareLoggedIn(handlerFollowing))
	c.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	c.register("browse", middlewareLoggedIn(handlerBrowse))
	if len(os.Args) < 2 {
		fmt.Print("No command given.\n")
		os.Exit(1)
	}
	cmd := command{
		name: os.Args[1],
		args: os.Args[2:],
	}
	if err = c.run(&s, cmd); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}