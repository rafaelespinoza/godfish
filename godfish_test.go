package godfish_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"bitbucket.org/rafaelespinoza/godfish"
)

const (
	baseTestOutputDir = "/tmp/godfish"
	testDBName        = "godfish_test"
)

// TODO: for each driver you want to test, check if a database exists. if it
// doesn't, then create one.
// if err = createTestDB(driver); err != nil {
// 	t.Errorf(
// 		"could not create test database for %q; %v",
// 		driver.Name(), err,
// 	)
// 	return
// }
func TestMain(m *testing.M) {
	os.MkdirAll(baseTestOutputDir, 0755)
	m.Run()
	os.RemoveAll(baseTestOutputDir)
}

func TestMigrationParams(t *testing.T) {
	var testDir *os.File
	var mig *godfish.MigrationParams
	var err error
	if testDir, err = os.Open(baseTestOutputDir); err != nil {
		t.Error(err)
		return
	}
	if mig, err = godfish.NewMigrationParams("foo", true, testDir); err != nil {
		t.Error(err)
		return
	}
	if mig.Forward.Direction() != godfish.DirForward {
		t.Errorf(
			"wrong Direction; expected %s, got %s",
			godfish.DirForward, mig.Forward.Direction(),
		)
	}
	if mig.Reverse.Direction() != godfish.DirReverse {
		t.Errorf(
			"wrong Direction; expected %s, got %s",
			godfish.DirReverse, mig.Reverse.Direction(),
		)
	}
	migrations := []godfish.Migration{mig.Forward, mig.Reverse}
	for _, mig := range migrations {
		if mig.Name() != "foo" {
			t.Errorf(
				"wrong Name; expected %s, got %s",
				"foo", mig.Name(),
			)
		}
		if mig.Timestamp().IsZero() {
			t.Error("got empty Timestamp")
		}
	}

	var filesBefore, filesAfter []string
	if filesBefore, err = testDir.Readdirnames(0); err != nil {
		t.Error(err)
		return
	}
	if err = mig.GenerateFiles(); err != nil {
		t.Error(err)
		return
	}
	if filesAfter, err = testDir.Readdirnames(0); err != nil {
		t.Error(err)
		return
	}
	if len(filesAfter)-len(filesBefore) != 2 {
		t.Errorf(
			"expected to generate 2 files, got %d",
			len(filesAfter)-len(filesBefore),
		)
		return
	}
	expectedDirections := []string{"reverse", "forward"}
	for i, name := range filesAfter {
		patt := fmt.Sprintf("[0-9]*.%s.foo.sql", expectedDirections[i])
		if match, err := filepath.Match(patt, name); err != nil {
			t.Error(err)
			return
		} else if !match {
			t.Errorf(
				"expected filename %q to match pattern %q",
				name, patt,
			)
		}
	}
}

func TestDriver(t *testing.T) {
	tests := []struct {
		driverName string
		dsnParams  godfish.DSNParams
	}{
		{
			driverName: "postgres",
			dsnParams: godfish.PostgresParams{
				Encoding: "UTF8",
				Host:     "localhost",
				Name:     testDBName,
				Pass:     os.Getenv("DB_PASSWORD"),
				Port:     "5432",
			},
		},
	}

	testDir, err := os.Open(baseTestOutputDir)
	if err != nil {
		t.Error(err)
		return
	}
	for i, test := range tests {
		driver, err := godfish.NewDriver(test.driverName, test.dsnParams)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() {
			if err := truncateSchemaMigrations(driver); err != nil {
				t.Errorf(
					"could not truncate schema_migrations table for %q; %v",
					driver.Name(), err,
				)
			}
		}()
		migrations, err := makeTestMigrations(driver.Name(), testDir)
		if err != nil {
			panic(err)
		}
		// test CreateSchemaMigrationsTable
		if err = godfish.CreateSchemaMigrationsTable(driver); err != nil {
			t.Errorf(
				"test [%d]; could not create schema migrations table for driver %q; %v",
				i, driver.Name(), err,
			)
		}

		// test ApplyMigration
		for j, mig := range migrations {
			err = godfish.ApplyMigration(
				driver,
				mig.Direction(),
				baseTestOutputDir,
				mig.Timestamp().Format(godfish.TimeFormat),
			)
			if err != nil {
				t.Errorf(
					"test [%d][%d]; driver %q; could not apply migration %v\n",
					i, j, driver.Name(), err,
				)
				return
			}
		}

		// test Migrate in forward direction
		if err = godfish.Migrate(driver, godfish.DirForward, baseTestOutputDir); err != nil {
			t.Errorf(
				"test [%d]; driver %q; could not Migrate in %s Direction",
				i, driver.Name(), godfish.DirForward,
			)
		}

		// test Info in forward direction
		fmt.Printf(
			"-- %s test [%d] calling Info %s %s\n",
			t.Name(), i, driver.Name(), godfish.DirForward,
		)
		if err = godfish.Info(driver, godfish.DirForward, baseTestOutputDir); err != nil {
			t.Errorf(
				"test [%d]; could not output info in %s Direction; %v",
				i, godfish.DirForward, err,
			)
			return
		}

		// test Info in reverse direction
		fmt.Printf(
			"-- %s test [%d] calling Info %s %s\n",
			t.Name(), i, driver.Name(), godfish.DirReverse,
		)
		if err = godfish.Info(driver, godfish.DirReverse, baseTestOutputDir); err != nil {
			t.Errorf(
				"test [%d]; could not output info in %s Direction; %v",
				i, godfish.DirReverse, err,
			)
			return
		}

		// test DumpSchema
		if err = godfish.DumpSchema(driver); err != nil {
			t.Errorf("test [%d]; could not dump schema %v", i, err)
			return
		}

		// test Migrate in reverse direction
		if err = godfish.Migrate(driver, godfish.DirReverse, baseTestOutputDir); err != nil {
			t.Errorf(
				"test [%d]; driver %q; could not Migrate in %s Direction",
				i, driver.Name(), godfish.DirReverse,
			)
		}
	}
}

// makeTestMigrations generates some stub migrations in both directions,
// generates the files and populates each with some dummy content. The
// migrations in the forward direction come before the reverse direction.
func makeTestMigrations(driverName string, testDir *os.File) ([]godfish.Migration, error) {
	type stubbedMigrationContent struct{ forward, reverse string }
	stubs := []stubbedMigrationContent{
		{
			forward: `CREATE TABLE foos (id int);`,
			reverse: `DROP TABLE foos;`,
		},
		{
			forward: `ALTER TABLE foos ADD COLUMN a varchar(255);`,
			reverse: `ALTER TABLE foos DROP COLUMN a;`,
		},
	}
	head := make([]godfish.Migration, 0)
	tail := make([]godfish.Migration, 0)
	for i, stub := range stubs {
		// need "unique" timestamps for migrations. TODO: think of workaround
		time.Sleep(1 * time.Second)

		var filename string
		var file *os.File
		var err error
		var params *godfish.MigrationParams
		defer func() {
			if file != nil {
				file.Close()
			}
		}()
		name := fmt.Sprintf("%s_%d", driverName, i)
		if params, err = godfish.NewMigrationParams(
			name,
			true,
			testDir,
		); err != nil {
			return nil, err
		}
		if err = params.GenerateFiles(); err != nil {
			return nil, err
		}

		for j, mig := range []godfish.Migration{params.Forward, params.Reverse} {
			if filename, err = godfish.Basename(mig); err != nil {
				return nil, err
			}
			if file, err = os.Open(baseTestOutputDir + "/" + filename); err != nil {
				return nil, err
			}
			// this only works if the slice we're iterating through has
			// migrations where each Direction is in the order:
			// [forward, reverse]
			if j == 0 {
				if file.WriteString(stub.forward); err != nil {
					return nil, err
				}
				head = append(head, mig)
				continue
			}
			if file.WriteString(stub.reverse); err != nil {
				return nil, err
			}
			tail = append(tail, mig)
		}
	}

	return append(head, tail...), nil
}

func truncateSchemaMigrations(driver godfish.Driver) (err error) {
	switch driver.Name() {
	case "postgres":
		cmd := exec.Command(
			"psql",
			testDBName, "-e", "-c", "TRUNCATE TABLE schema_migrations CASCADE",
		)
		_, err = cmd.Output()
		if val, ok := err.(*exec.ExitError); ok {
			fmt.Println(string(val.Stderr))
			err = val
		}
	default:
		err = fmt.Errorf("unknown Driver %q", driver.Name())
	}
	return
}

func createTestDB(driver godfish.Driver) (err error) {
	switch driver.Name() {
	case "postgres":
		dsnParams := driver.DSNParams()
		var params godfish.PostgresParams
		if dsn, ok := dsnParams.(godfish.PostgresParams); !ok {
			err = fmt.Errorf("expected dsnParams to be a PostgresParams, got %T", dsn)
			return
		} else {
			params = dsn
		}
		cmd := exec.Command(
			"createdb",
			"-e",
			"--encoding", params.Encoding,
			"--host", params.Host,
			"--port", params.Port,
			"--username", params.User,
			"--no-password",
			params.Name,
		)
		_, err = cmd.Output()
		if val, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("%v. %v", val.String(), string(val.Stderr))
		}
	default:
		err = fmt.Errorf("unknown Driver %q", driver.Name())
	}
	return
}

func dropTestDB(driver godfish.Driver) (err error) {
	switch driver.Name() {
	case "postgres":
		dsnParams := driver.DSNParams()
		var params godfish.PostgresParams
		if dsn, ok := dsnParams.(godfish.PostgresParams); !ok {
			err = fmt.Errorf("expected dsnParams to be a PostgresParams, got %T", dsn)
			return
		} else {
			params = dsn
		}
		cmd := exec.Command(
			"dropdb",
			"-e", "--if-exists",
			"--host", params.Host,
			"--port", params.Port,
			"--username", params.User,
			"--no-password",
			params.Name,
		)
		_, err = cmd.Output()
		if val, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("%v. %v", val.String(), string(val.Stderr))
		}
	default:
		err = fmt.Errorf("unknown Driver %q", driver.Name())
	}
	return
}
