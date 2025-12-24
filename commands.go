package main

import (
	"github.com/Baehry/gator/internal/config"
	"github.com/Baehry/gator/internal/database"
	"context"
	"github.com/google/uuid"
	"time"
	"fmt"
	"strconv"
)

type state struct {
	db *database.Queries
	cfgPtr *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	commands map[string]func(*state, command) error
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		current_user, err := s.db.GetUser(context.Background(), s.cfgPtr.Current_user_name)
		if err != nil {
			return err
		}
		return handler(s, cmd, current_user)
	}
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("No arguments given.")
	}
	_, err := s.db.GetUser(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("User '%v' does not exist.", cmd.args[0])
	}
	if err := s.cfgPtr.SetUser(cmd.args[0]); err != nil {
		return err
	}
	fmt.Printf("user has been set as '%s'.\n", cmd.args[0])
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("No arguments given.")
	}
	params := database.CreateUserParams{
		ID: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name: cmd.args[0],
	}
	_, err := s.db.CreateUser(context.Background(), params)
	if err != nil {
		return fmt.Errorf("User '%v' already exists.", cmd.args[0])
	}
	if err := s.cfgPtr.SetUser(cmd.args[0]); err != nil {
		return err
	}
	fmt.Printf("user '%v' successfully registered.\n", cmd.args[0])
	return nil
}

func handlerReset(s *state, cmd command) error {
	if err := s.db.ResetUsers(context.Background()); err != nil {
		return err
	}
	fmt.Print("All users successfully removed.\n")
	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err
	}
	for _, user := range users {
		fmt.Print(user.Name)
		if user.Name == s.cfgPtr.Current_user_name {
			fmt.Print(" (current)")
		}
		fmt.Print("\n")
	}
	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("No argument given.")
	}
	duration, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return fmt.Errorf("Invalid duration.")
	}
	fmt.Printf("collecting every %v.\n", duration)
	ticker := time.NewTicker(duration)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 2 {
		return fmt.Errorf("Not enough arguments.")
	}
	params := database.CreateFeedParams{
		ID: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name: cmd.args[0],
		Url: cmd.args[1],
		UserID: user.ID,
	}
	feed, err := s.db.CreateFeed(context.Background(), params)
	if err != nil {
		return fmt.Errorf("Feed with  url '%v' already exists.", cmd.args[1])
	}
	feedParams := database.CreateFeedFollowParams{
		ID: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID: user.ID,
		FeedID: feed.ID,
	}
	_, err = s.db.CreateFeedFollow(context.Background(), feedParams)
	if err != nil {
		return err
	}
	fmt.Printf("Feed '%v' successfully Added.\n", cmd.args[0])
	return nil
}

func handlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return err
	}
	for _, feed := range feeds {
		user, err := s.db.GetUserById(context.Background(), feed.UserID)
		if err != nil {
			return err
		}
		fmt.Printf("'%s' by %s: %s\n", feed.Name, user.Name, feed.Url)
	}
	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("Not argument given.")
	}
	feed, err := s.db.GetFeed(context.Background(), cmd.args[0])
	if err != nil {
		return err
	}
	params := database.CreateFeedFollowParams{
		ID: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID: user.ID,
		FeedID: feed.ID,
	}
	follow, err := s.db.CreateFeedFollow(context.Background(), params)
	if err != nil {
		return fmt.Errorf("%s is already following '%s'.", user.Name, feed.Name)
	}
	fmt.Printf("'%s' was followed by %s.\n", follow.FeedName, follow.UserName)
	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return err
	}
	for _, follow := range follows {
		fmt.Printf("%s\n", follow.FeedName)
	}
	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("Not argument given.")
	}
	feed, err := s.db.GetFeed(context.Background(), cmd.args[0])
	if err != nil {
		return err
	}
	params := database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	}
	if err := s.db.DeleteFeedFollow(context.Background(), params); err != nil {
		return fmt.Errorf("%s is not following '%s'.", user.Name, feed.Name)
	}
	fmt.Printf("'%s' was unfollowed by %s.\n", feed.Name, user.Name)
	return nil
}

func handlerBrowse(s *state, cmd command, user database.User) error {
	limit := 2
	var err error
	if len(cmd.args) > 0 {
		limit, err = strconv.Atoi(cmd.args[0])
		if err != nil {
			return err
		}
	}
	params := database.GetPostsForUserParams{
		UserID: user.ID,
		Limit: int32(limit),
	}
	posts, err := s.db.GetPostsForUser(context.Background(), params)
	if err != nil {
		return err
	}
	for _, post := range posts {
		fmt.Printf("%s\n  %s\n\n", post.Title, post.Description)
	}
	return nil
}

func (c *commands) run(s *state, cmd command) error {
	command, ok := c.commands[cmd.name]
	if !ok {
		return fmt.Errorf("command '%s' not found", cmd.name)
	}
	return command(s, cmd)
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.commands[name] = f
}