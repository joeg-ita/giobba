# Testing Giobba

Run the test suite using the following commands:

```sh
# Execute all tests
go test ./test/...

# Execute individual test files
go test ./test/test_helper.go ./test/datetime_test.go
go test ./test/test_helper.go ./test/giobba_test.go
go test ./test/test_helper.go ./test/job_test.go
go test ./test/test_helper.go ./test/main_task_test.go
go test ./test/test_helper.go ./test/priority_test.go
go test ./test/test_helper.go ./test/revoke_test.go
go test ./test/test_helper.go ./test/test_helper.go
```