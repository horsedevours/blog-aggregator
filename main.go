package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/horsedevours/blog-aggregator/internal/config"
	"github.com/horsedevours/blog-aggregator/internal/database"
	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		fmt.Printf("error when reading config: %v\n", err)
	}

	st := state{cfg: &cfg}
	db, err := sql.Open("postgres", st.cfg.DbUrl)
	if err != nil {
		fmt.Printf("error opening DB connection: %v", err)
		os.Exit(1)
	}
	st.db = database.New(db)

	cmds := commands{cmdMap: map[string]func(*state, command) error{}}
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("addfeed", handlerAddfeed)
	cmds.register("feeds", handlerFeeds)
	cmds.register("follow", handlerFollow)
	cmds.register("following", handlerFollowing)

	if len(os.Args) < 2 {
		fmt.Println("at least 2 args required")
		os.Exit(1)
	}

	var cmd command
	cmd.name = os.Args[1]
	if len(os.Args) > 2 {
		cmd.args = os.Args[2:]
	}

	err = cmds.run(&st, cmd)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}
