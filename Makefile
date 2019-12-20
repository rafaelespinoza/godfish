TEST_DB_NAME=godfish_test
DB_USER=godfish
DB_HOST=localhost

# Register database drivers to test. For every item in this array, there should
# be three separate targets elsewhere in the Makefile. Here's an example using a
# made-up DBMS:
#
#	foodb-setup:
#		command to create $(TEST_DB_NAME)
#	foodb-teardown:
#		command to drop $(TEST_DB_NAME)
#	foodb: foodb-setup foodb-teardown
#
# One should have a target suffix "-teardown", another should have the target
# suffix "-setup" and the last one is just named after the DBMS, but it targets
# the other two.
DRIVERS_TO_TEST=postgres mysql

SETUPS=$(addsuffix -setup, $(DRIVERS_TO_TEST))
TEARDOWNS=$(addsuffix -teardown, $(DRIVERS_TO_TEST))

test: clean db
	go test ./godfish $(ARGS)

install:
	go install -i .

db: $(SETUPS)
clean: $(TEARDOWNS)

postgres: postgres-teardown postgres-setup
postgres-teardown:
	dropdb --if-exists $(TEST_DB_NAME)
postgres-setup:
	createdb -E utf8 $(TEST_DB_NAME)

mysql: mysql-teardown mysql-setup
mysql-teardown:
	mysql -u $(DB_USER) -h $(DB_HOST) \
		-e "DROP DATABASE IF EXISTS ${TEST_DB_NAME}"
mysql-setup:
	mysql -u $(DB_USER) -h $(DB_HOST) \
		-e "CREATE DATABASE IF NOT EXISTS ${TEST_DB_NAME}"
