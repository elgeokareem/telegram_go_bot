## Bot Telegram

This is a Telegram bot written in Go.

### Migrations

This project uses `golang-migrate/migrate` to manage database schema changes.

**1. Create a new migration:**

Use the `make migrate-create` command. You must provide a descriptive name for your migration.

```sh
make migrate-create name=add_new_feature
```

This will create two new files in `database/migrations` with the next sequential version number (e.g., `000002_add_new_feature.up.sql` and `000002_add_new_feature.down.sql`).

**2. Run migrations:**

To apply all pending `up` migrations:

```sh
make migrate-up
```

**3. Revert migrations:**

To revert the most recently applied migration:

```sh
make migrate-down
```

These commands use the `migrate.go` file and do not require the `migrate` CLI to be installed on every machine, making them portable for other developers.
