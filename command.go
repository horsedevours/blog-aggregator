package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/horsedevours/blog-aggregator/internal/database"
	"github.com/lib/pq"
)

type command struct {
	name string
	args []string
}

type commands struct {
	cmdMap map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	err := c.cmdMap[cmd.name](s, cmd)
	return err
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.cmdMap[name] = f
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("missing username argument\n")
	}

	_, err := s.db.GetUser(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("user does not exist")
	}

	s.cfg.SetUser(cmd.args[0])
	fmt.Printf("username set to: %s\n", cmd.args[0])
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("register requires a name argument\n")
	}

	user, _ := s.db.GetUser(context.Background(), cmd.args[0])
	if user.Name == cmd.args[0] {
		return fmt.Errorf("user %s already exists", cmd.args[0])
	}

	user, err := s.db.CreateUser(context.Background(), database.CreateUserParams{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), Name: cmd.args[0]})
	if err != nil {
		return fmt.Errorf("error creating user: %v\n", err)
	}

	err = s.cfg.SetUser(cmd.args[0])
	if err != nil {
		return fmt.Errorf("unable to update current user: %v\n", err)
	}

	fmt.Printf("user successfully created: %v\n", user)
	return nil
}

func handlerReset(s *state, cmd command) error {
	err := s.db.DeleteAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error deleting all users: %v\n", err)
	}

	fmt.Println("all user records deleted")
	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error querying users: %v\n", err)
	}

	for _, user := range users {
		if user.Name == s.cfg.CurrentUserName {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}
	return nil
}

func handlerAgg(s *state, cmd command, timeBetweenReqs time.Duration) error {
	user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
	if err != nil {
		return err
	}

	fmt.Printf("Collecting feeds every %s\n", timeBetweenReqs.String())
	ticker := time.NewTicker(timeBetweenReqs)
	for ; ; <-ticker.C {
		scrapeFeeds(s, user.ID)
	}
}

func scrapeFeeds(s *state, id uuid.UUID) {
	feed, err := s.db.GetNextFeedToFetch(context.Background(), id)
	if err != nil {
		fmt.Printf("error getting next feed from database: %v", err)
		return
	}

	s.db.MarkFeedFetched(context.Background(), database.MarkFeedFetchedParams{
		LastFetchedAt: sql.NullTime{Time: time.Now(), Valid: true},
		ID:            feed.ID,
	})

	rss, err := fetchFeed(context.Background(), feed.Url)
	if err != nil {
		fmt.Printf("error fetching feed from web: %v", err)
	}

	for _, item := range rss.Channel.Item {
		fmt.Printf("The date format: %s\n", item.PubDate)
		publishedAt := sql.NullTime{}

		if pubDate, err := time.Parse(time.RFC1123Z, item.PubDate); err != nil {
			fmt.Printf("error parsing publish date for %s; saving as NULL", item.Title)
		} else {
			publishedAt.Time = pubDate
			publishedAt.Valid = true
		}
		_, err = s.db.CreatePost(context.Background(), database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       item.Title,
			Url:         item.Link,
			Description: sql.NullString{String: item.Description},
			PublishedAt: publishedAt,
			FeedID:      feed.ID,
		})

		if pqerr, ok := err.(*pq.Error); ok {
			if pqerr.Code.Name() == "unique_violation" {
				continue
			}
		} else if err != nil {
			fmt.Printf("unexpected error: %v\n", err)
		}
	}
}

func handlerAddfeed(s *state, cmd command, user database.User) error {
	name := cmd.args[0]
	url := cmd.args[1]

	feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	})
	if err != nil {
		return fmt.Errorf("error creating feed: %w\n", err)
	}

	_, err = s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("error creating follow: %w", err)
	}

	fmt.Printf("%v", feed)
	return nil
}

func handlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("error getting feeds: %w\n", err)
	}

	fmt.Println("All o' y'all feeds:")
	for _, feed := range feeds {
		fmt.Printf(" - %s | %s | User: %s\n", feed.Name, feed.Url, feed.Name_2.String)
	}
	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	url := cmd.args[0]

	feed, err := s.db.GetFeed(context.Background(), url)
	if err != nil {
		return fmt.Errorf("error getting feed: %w", err)
	}

	follow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("error creating follow: %w", err)
	}

	fmt.Printf("User %s is now following feed %s\n", follow.UserName, follow.FeedName)
	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	following, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("error getting follows: %w\n", err)
	}

	for _, follow := range following {
		fmt.Printf("* %s\n", follow.FeedName)
	}
	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	feed, err := s.db.GetFeed(context.Background(), cmd.args[0])
	if err != nil {
		return err
	}

	err = s.db.UnfollowFeed(context.Background(), database.UnfollowFeedParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return nil
	}

	return nil
}

func handlerBrowse(s *state, cmd command, user database.User) error {
	limit, err := getPostLimit(cmd)
	if err != nil {
		return err
	}

	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  limit,
	})
	if err != nil {
		return err
	}

	for i, post := range posts {
		fmt.Printf("%d. %s\n", i+1, post.Title)
		if post.Description.Valid {
			fmt.Printf("  * %s\n", post.Description.String)
		}
		fmt.Printf("    %s\n", post.Url)
		if post.PublishedAt.Valid {
			fmt.Printf("  Published: %s\n", post.PublishedAt.Time)
		}
	}

	return nil
}

func getPostLimit(cmd command) (int32, error) {
	if len(cmd.args) > 0 {
		limit, err := strconv.Atoi(cmd.args[0])
		if err != nil {
			return 0, err
		}
		return int32(limit), nil
	}
	return 2, nil
}
