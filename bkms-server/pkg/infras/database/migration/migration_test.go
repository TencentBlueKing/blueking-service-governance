package migration

import (
	"encoding/json"
	"io/fs"

	"github.com/bytedance/mockey"
	gomigrate "github.com/golang-migrate/migrate/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/db"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

const (
	// testFirstMigrationVersion is the sequence number of the first embedded migration, always present.
	testFirstMigrationVersion = uint(1)
	// testBinaryMaxVersion is the largest version the binary embeds in Up branch tests.
	testBinaryMaxVersion = uint(3)
	// testNewerDBVersion simulates a database migrated by a newer binary.
	testNewerDBVersion = uint(4)
	// testNilDBVersion is the placeholder version reported together with gomigrate.ErrNilVersion.
	testNilDBVersion = uint(0)
	// testRunNotCalled means Up should skip before delegating to run.
	testRunNotCalled = 0
	// testRunCalledOnce means Up should keep the original migration path.
	testRunCalledOnce = 1
)

var _ = Describe("Database migration", func() {
	Describe("mongoCfgToMigrateURL", func() {
		It("escapes credentials and includes the configured database", func() {
			cfg := config.MongoConfig{
				Username: "user@example.com",
				Password: "p@ss:/? #",
				Host:     "2001:db8::1",
				Port:     "27017",
				Database: "test db",
			}

			Expect(mongoCfgToMigrateURL(cfg)).To(Equal(
				"mongodb://user%40example.com:p%40ss%3A%2F%3F%20%23@[2001:db8::1]:27017/test%20db?authSource=admin",
			))
		})

		It("omits user info when no username is configured", func() {
			cfg := config.MongoConfig{Host: "mongo", Port: "27017", Database: "bkms"}
			Expect(mongoCfgToMigrateURL(cfg)).To(Equal("mongodb://mongo:27017/bkms?authSource=admin"))
		})
	})

	Describe("maxEmbeddedMigrationVersion", func() {
		It("returns at least the first embedded up migration version", func() {
			version, err := maxEmbeddedMigrationVersion()
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(BeNumerically(">=", testFirstMigrationVersion))
		})
	})

	Describe("embeddedMigrationVersion", func() {
		It("parses the sequence number prefix", func() {
			version, err := embeddedMigrationVersion("migrations/000005_some_model_idx.up.json")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(uint(5)))
		})

		It("returns error for invalid migration file name", func() {
			_, err := embeddedMigrationVersion("migrations/invalid.up.json")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Up", func() {
		It("skips migration when the switch is on and database version is newer than binary version", func() {
			expectUpRunCalls(true, testNewerDBVersion, false, nil, testRunNotCalled)
		})

		It("keeps migrating when the switch is off, even for a newer database version", func() {
			expectUpRunCalls(false, testNewerDBVersion, false, nil, testRunCalledOnce)
		})

		It("keeps migrating for a dirty database so the failed migration is not hidden", func() {
			expectUpRunCalls(true, testNewerDBVersion, true, nil, testRunCalledOnce)
		})

		It("keeps migrating when database has no version", func() {
			expectUpRunCalls(true, testNilDBVersion, false, gomigrate.ErrNilVersion, testRunCalledOnce)
		})
	})

	It("embeds valid JSON migration files", func() {
		paths, err := fs.Glob(db.Migrations, "migrations/*.json")
		Expect(err).NotTo(HaveOccurred())
		Expect(paths).NotTo(BeEmpty())

		for _, path := range paths {
			content, readErr := fs.ReadFile(db.Migrations, path)
			Expect(readErr).NotTo(HaveOccurred())
			var commands []map[string]any
			Expect(json.Unmarshal(content, &commands)).To(Succeed(), path)
		}
	})
})

// expectUpRunCalls verifies whether Up delegates to run after checking database and binary versions.
// It keeps migration branch tests focused on behavior instead of repeating mockey setup details.
func expectUpRunCalls(allowSkipNewer bool, dbVersion uint, dirty bool, versionErr error, wantRunCalled int) {
	mockey.PatchConvey("check up run calls", GinkgoT(), func() {
		m := &Migration{migrate: &gomigrate.Migrate{}, allowSkipNewer: allowSkipNewer}
		runCalled := 0
		mockey.Mock((*gomigrate.Migrate).Version).Return(dbVersion, dirty, versionErr).Build()
		mockey.Mock(maxEmbeddedMigrationVersion).Return(testBinaryMaxVersion, nil).Build()
		mockey.Mock((*Migration).run).To(func(_ *Migration, _ string, _ func() error) error {
			runCalled++
			return nil
		}).Build()

		Expect(m.Up(nil)).To(Succeed())
		Expect(runCalled).To(Equal(wantRunCalled))
	})
}
