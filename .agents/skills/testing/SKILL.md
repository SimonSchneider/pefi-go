# Testing

Testing should focus mainly on Models (service layer). It's important only to excercise the public API used by Controllers.

Tests should be in a `_test` package. ie.

the package
```
package service
```
should be tested in the same directory but tests should have the package
```
package service_test
```

**Do not export APIs only for testing, use the public API excercised by the Controllers only.**

Tests should mainly be integration tests with real DB and as close as possible to the real world with no mocking and no partial tests, they should excercise the models just like the controllers do.
