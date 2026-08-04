package cmd

import (
	"context"
	stderrors "errors"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	dbmigration "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database/migration"
)

// See https://pkg.go.dev/github.com/golang-migrate/migrate/v4#section-readme for details.
type databaseMigrator interface {
	Goto(version uint) error
	Up(limit *int) error
	Down(limit *int) error
	Force(version int) error
	Close() error
}

type databaseMigratorFactory func(context.Context, string) (databaseMigrator, error)

// NewMigrateCmd creates the parent command for application database migrations.
func NewMigrateCmd() *cobra.Command {
	return newMigrateCmd(openDatabaseMigrator)
}

func newMigrateCmd(factory databaseMigratorFactory) *cobra.Command {
	var srvCfg string

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Manage application database migrations",
	}
	cmd.PersistentFlags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	_ = cmd.MarkPersistentFlagRequired("srvCfg")

	cmd.AddCommand(newMigrateGotoCmd(&srvCfg, factory))
	cmd.AddCommand(newMigrateUpCmd(&srvCfg, factory))
	cmd.AddCommand(newMigrateDownCmd(&srvCfg, factory))
	cmd.AddCommand(newMigrateForceCmd(&srvCfg, factory))
	return cmd
}

func newMigrateGotoCmd(srvCfg *string, factory databaseMigratorFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "goto V",
		Short: "Migrate to version V",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version, err := strconv.ParseUint(args[0], 10, strconv.IntSize)
			if err != nil {
				return errors.Wrap(err, "parse migration version V")
			}
			return withDatabaseMigrator(cmd.Context(), *srvCfg, factory, func(migrator databaseMigrator) error {
				return migrator.Goto(uint(version))
			})
		},
	}
}

func newMigrateUpCmd(srvCfg *string, factory databaseMigratorFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "up [N]",
		Short: "Apply all or N up migrations",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, err := migrationLimit(args)
			if err != nil {
				return err
			}
			return withDatabaseMigrator(cmd.Context(), *srvCfg, factory, func(migrator databaseMigrator) error {
				return migrator.Up(limit)
			})
		},
	}
}

func newMigrateDownCmd(srvCfg *string, factory databaseMigratorFactory) *cobra.Command {
	var applyAll bool

	cmd := &cobra.Command{
		Use:   "down [N]",
		Short: "Apply all or N down migrations",
		Long:  "Apply N down migrations. Use --all to apply all down migrations.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if applyAll && len(args) != 0 {
				return errors.New("--all cannot be used with N")
			}
			if !applyAll && len(args) == 0 {
				return errors.New("either N or --all must be specified")
			}

			limit, err := migrationLimit(args)
			if err != nil {
				return err
			}

			return withDatabaseMigrator(cmd.Context(), *srvCfg, factory, func(migrator databaseMigrator) error {
				return migrator.Down(limit)
			})
		},
	}
	cmd.Flags().BoolVar(&applyAll, "all", false, "apply all down migrations")
	return cmd
}

func newMigrateForceCmd(srvCfg *string, factory databaseMigratorFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "force V",
		Short: "Set version V but don't run migration (ignores dirty state)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version, err := strconv.ParseInt(args[0], 10, strconv.IntSize)
			if err != nil {
				return errors.Wrap(err, "parse migration version V")
			}
			return withDatabaseMigrator(cmd.Context(), *srvCfg, factory, func(migrator databaseMigrator) error {
				return migrator.Force(int(version))
			})
		},
	}
}

func openDatabaseMigrator(ctx context.Context, srvCfg string) (databaseMigrator, error) {
	cfg, err := config.Load(ctx, srvCfg)
	if err != nil {
		return nil, errors.Wrap(err, "load config")
	}
	if err = log.InitDefaultLogger(cfg.Logging); err != nil {
		return nil, errors.Wrap(err, "init logger")
	}

	migrator, err := dbmigration.New(ctx, cfg.Mongo, cfg.Development.AllowSkipNewerDBMigration)
	if err != nil {
		return nil, err
	}
	return migrator, nil
}

func withDatabaseMigrator(
	ctx context.Context,
	srvCfg string,
	factory databaseMigratorFactory,
	run func(databaseMigrator) error,
) (err error) {
	migrator, err := factory(ctx, srvCfg)
	if err != nil {
		return err
	}
	defer func() {
		err = stderrors.Join(err, errors.Wrap(migrator.Close(), "close migration runner"))
	}()
	return run(migrator)
}

// migrationLimit 根据传入的命令行参数获取执行 up 或 down 命令时的 limit 值（最大次数）
func migrationLimit(args []string) (*int, error) {
	if len(args) == 0 {
		return nil, nil
	}

	limit, err := strconv.Atoi(args[0])
	if err != nil {
		return nil, errors.Wrap(err, "parse migration count N")
	}
	if limit < 0 {
		return nil, errors.New("migration count N must not be negative")
	}
	return &limit, nil
}
