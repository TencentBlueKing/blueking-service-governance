// Package migration runs the application's embedded MongoDB migrations.
package migration

import (
	"context"
	stderrors "errors"
	"io/fs"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"

	gomigrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mongodb"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/db"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

const (
	// migrationsPath is the root directory embedded by db.Migrations and consumed by golang-migrate.
	migrationsPath = "migrations"
	// migrationUpFileSuffix limits binary version detection to executable up migration files.
	migrationUpFileSuffix = ".up.json"
	// migrationVersionSep separates the leading numeric version from the migration description.
	migrationVersionSep = "_"
	// migrationVersionBase parses migration versions as decimal sequence numbers from file names.
	migrationVersionBase = 10
	// migrationVersionBitSize keeps parsed versions assignable to uint on every supported platform.
	migrationVersionBitSize = 32
)

// Migration wraps golang-migrate with the operations exposed by bkms-server.
type Migration struct {
	ctx      context.Context
	database string
	migrate  *gomigrate.Migrate
	// allowSkipNewer tolerates a database migrated by a newer binary, see config.DevConfig.AllowSkipNewerDBMigration.
	allowSkipNewer bool
}

// New initializes a migration runner from the embedded migration files and the
// MongoDB address in the application configuration.
// allowSkipNewer comes from config.DevConfig.AllowSkipNewerDBMigration and only affects Up.
func New(ctx context.Context, cfg config.MongoConfig, allowSkipNewer bool) (*Migration, error) {
	sourceDriver, err := iofs.New(db.Migrations, migrationsPath)
	if err != nil {
		return nil, errors.Wrap(err, "initialize embedded migration source")
	}

	databaseDriver, err := (&mongodb.Mongo{}).Open(mongoCfgToMigrateURL(cfg))
	if err != nil {
		_ = sourceDriver.Close()
		return nil, errors.Wrap(err, "open MongoDB migration database")
	}

	runner, err := gomigrate.NewWithInstance("iofs", sourceDriver, "mongodb", databaseDriver)
	if err != nil {
		return nil, stderrors.Join(
			errors.Wrap(err, "initialize migration runner"),
			sourceDriver.Close(),
			databaseDriver.Close(),
		)
	}
	runner.Log = migrateLogger{ctx: ctx}

	return &Migration{ctx: ctx, database: cfg.Database, migrate: runner, allowSkipNewer: allowSkipNewer}, nil
}

// Close releases the migration source and MongoDB connection.
func (m *Migration) Close() error {
	sourceErr, databaseErr := m.migrate.Close()
	return stderrors.Join(sourceErr, databaseErr)
}

// Goto migrates the database to an exact version.
func (m *Migration) Goto(version uint) error {
	return m.run("goto", func() error { return m.migrate.Migrate(version) })
}

// Up applies all pending migrations, or at most limit migrations when limit is non-nil.
func (m *Migration) Up(limit *int) error {
	skipped, err := m.skipNewerDBVersion()
	if err != nil {
		return errors.Wrap(err, "check migration version before up")
	}
	if skipped {
		return nil
	}
	if limit == nil {
		return m.run("up", m.migrate.Up)
	}
	return m.run("up", func() error { return m.migrate.Steps(*limit) })
}

// Down applies all down migrations, or at most limit migrations when limit is non-nil.
func (m *Migration) Down(limit *int) error {
	if limit == nil {
		return m.run("down", m.migrate.Down)
	}
	return m.run("down", func() error { return m.migrate.Steps(-*limit) })
}

// Force sets the migration version without running migrations and clears dirty state.
func (m *Migration) Force(version int) error {
	return m.run("force", func() error { return m.migrate.Force(version) })
}

// UpAll opens a migration runner, applies every pending up migration, and closes it.
func UpAll(ctx context.Context, cfg config.MongoConfig, allowSkipNewer bool) (err error) {
	runner, err := New(ctx, cfg, allowSkipNewer)
	if err != nil {
		return err
	}
	defer func() {
		err = stderrors.Join(err, errors.Wrap(runner.Close(), "close migration runner"))
	}()

	return runner.Up(nil)
}

func (m *Migration) run(operation string, run func() error) error {
	attrs := []slog.Attr{
		slog.String("operation", operation),
		slog.String("database", m.database),
	}
	log.InfoAttrs(m.ctx, "running database migration", attrs...)

	err := run()
	if errors.Is(err, gomigrate.ErrNoChange) {
		log.InfoAttrs(m.ctx, "database migration has no changes", attrs...)
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "run database migration %s", operation)
	}

	log.InfoAttrs(m.ctx, "database migration completed", attrs...)
	return nil
}

// skipNewerDBVersion reports whether Up should silently skip because the database was
// already migrated by a newer binary, which makes golang-migrate fail on the unknown version.
// Skipping requires the AllowSkipNewerDBMigration switch and never applies to a dirty database,
// because a dirty state means a migration failed and must not be hidden.
func (m *Migration) skipNewerDBVersion() (bool, error) {
	if !m.allowSkipNewer {
		return false, nil
	}

	dbVersion, dirty, err := m.migrate.Version()
	if err != nil {
		if errors.Is(err, gomigrate.ErrNilVersion) {
			return false, nil
		}
		return false, errors.Wrap(err, "get database migration version")
	}
	if dirty {
		return false, nil
	}

	binaryMaxVersion, err := maxEmbeddedMigrationVersion()
	if err != nil {
		return false, err
	}
	if dbVersion <= binaryMaxVersion {
		return false, nil
	}

	log.WarnAttrs(m.ctx, "database migration version is newer than binary, skip migration",
		slog.Uint64("db_version", uint64(dbVersion)),
		slog.Uint64("binary_max_version", uint64(binaryMaxVersion)),
		slog.String("database", m.database),
	)
	return true, nil
}

// maxEmbeddedMigrationVersion returns the largest version from embedded up migration files.
// Only up files are considered because they define the highest schema version this binary can apply.
func maxEmbeddedMigrationVersion() (uint, error) {
	paths, err := fs.Glob(db.Migrations, migrationsPath+"/*"+migrationUpFileSuffix)
	if err != nil {
		return 0, errors.Wrap(err, "find embedded up migration files")
	}
	if len(paths) == 0 {
		return 0, errors.New("no embedded up migration files found")
	}

	var maxVersion uint
	for _, path := range paths {
		version, err := embeddedMigrationVersion(path)
		if err != nil {
			return 0, err
		}
		if version > maxVersion {
			maxVersion = version
		}
	}
	return maxVersion, nil
}

// embeddedMigrationVersion parses the numeric version prefix from an embedded migration file path.
// File names must follow the golang-migrate convention: <version>_<description>.up.json,
// where version is the zero-padded sequence number produced by `migrate create -seq`.
func embeddedMigrationVersion(path string) (uint, error) {
	fileName := strings.TrimPrefix(path, migrationsPath+"/")
	versionText, _, found := strings.Cut(fileName, migrationVersionSep)
	if !found {
		return 0, errors.Errorf("parse embedded migration version from %s", path)
	}
	version, err := strconv.ParseUint(versionText, migrationVersionBase, migrationVersionBitSize)
	if err != nil {
		return 0, errors.Wrapf(err, "parse embedded migration version from %s", path)
	}
	return uint(version), nil
}

// mongoCfgToMigrateURL includes the database in the path because the migrate MongoDB
// driver requires it. authSource remains admin to preserve the authentication
// behavior of config.Mongo.GetURI(), whose URI has no database path.
func mongoCfgToMigrateURL(cfg config.MongoConfig) string {
	mongoURL := url.URL{
		Scheme: "mongodb",
		Host:   net.JoinHostPort(cfg.Host, cfg.Port),
		Path:   "/" + cfg.Database,
	}
	if cfg.Username != "" {
		mongoURL.User = url.UserPassword(cfg.Username, cfg.Password)
	}
	query := mongoURL.Query()
	query.Set("authSource", "admin")
	mongoURL.RawQuery = query.Encode()
	return mongoURL.String()
}

type migrateLogger struct {
	ctx context.Context
}

func (l migrateLogger) Printf(format string, args ...any) {
	log.Debugf(l.ctx, strings.TrimSuffix(format, "\n"), args...)
}

func (migrateLogger) Verbose() bool {
	return true
}

var _ gomigrate.Logger = migrateLogger{}
