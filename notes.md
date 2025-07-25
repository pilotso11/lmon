# Notes

## Architecture
* Web server backend
* Monitors implemented as configuration components
* Monitor implementations are independent of the web server and all implemnent a common interface
* Web server provides a REST API for monitor implementations to register themselves
* Web server provides a REST API for clients to query monitor status
* The web server serves up a largely static web application using go templates for SSR of everything but inteactive UI components.
* Each UI component is a separate web template for easy maintenance and development
* There is no use of JavaScript frameworks like React or Vue, as the goal is to keep the application simple and lightweight.
* The web server is written in Go, leveraging its concurrency model and performance.
* THe web server used the standard library's `net/http` package for handling HTTP requests and responses.
* The web server uses `html/template` for rendering HTML templates, allowing for dynamic content generation.
* The web server uses `gorilla/mux` for routing, providing a flexible way to define routes and handle parameters.
* The web server uses go:embed for serving static files, allowing for easy inclusion of CSS, images, and other assets.
* User configuration is stored in a simple JSON file, making it easy to edit and maintain.
* Configuration can be overridden by environment variables, allowing for flexibility in deployment using viper.

## Implementation Goals
* Keep the implementation simple and lightweight
* Use Go's standard library as much as possible
* Avoid complex dependencies
* Use a single binary for deployment
* Make it easy to extend with new monitor implementations
* Extensive use of unit tests, integration tests, and end-to-end tests with a goal of 90% test coverage, and the only gaps being hard to procuce exception cases.
* Use of testing libraries like `testify` for assertions and mocking. Attention payed to timeouts using assert.Eventually(), as well as asycnh issues, test -race is used for validation.
* Full testing of the UI using rod. 
* Rod tests avoid the use of Must functions that panic on failure, instead using error handling to allow for graceful test failures.
* New implementions follow the same patterns and practices as existing implementations.
