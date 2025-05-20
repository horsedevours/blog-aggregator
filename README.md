Requires
- Go
- PostgreSQL

Run `go install` at root to compile.

Configuration
- Create .gatorconfig.json in home directory. Contents should include your Postgres connection string and a field for user name.
  ```
  {
    "db_url":"postgres://<username>:<password>@<host>:<port>/gator?sslmode=disable",
    "current_user_name":""
  }
  ```
- Register your username first: `blog-aggregator <username>`

Commands
  login <username> - Change user
  users - List all users
  addfeed <name> <url> - Add an RSS feed to track (will automatically follow for the logged-in user)
  feeds - List all feeds
  follow <url> - Follow an existing feed
  following - List all feeds followed by the logged-in user
  unfollow <url> - Removes feed from current user's follows (feed still exists in database for following)
  agg <time-between-requests> - Starts service that fetches RSS feed data at the interval designated by user (formt: 1s = every second, 5m = every 5m, 10h = every 10h, etc.)
  browse <limit> - Prints a list to console of the most recent RSS items for feeds followed by the current user. Limit specifies how many records to display.
