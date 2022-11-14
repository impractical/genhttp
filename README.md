# genhttp

`genhttp` provides a generics-powered HTTP microframework for handling requests.

The genhttp microframework is really a small pattern around request handling.
The pattern is that a request will be parsed, then validated, then executed.
This is represented in code by making each endpoint a type, and giving them a
`ParseRequest`, `ValidateRequest`, and `ExecuteRequest` method. A `Handle`
helper function calls these functions in order, checking for errors in between
and short-circuiting when they're found. Thanks to generics, these methods can
operate on an endpoint-specific request type that gets created by the
`ParseRequest` method and passed directly into subsequent methods.

This pattern is useful for doing unit testing, as it separates the three
responsibilities of an HTTP endpoint into discrete functions and makes them
accessible through a more constrained interface than an `http.Request` and a
more easily-examined interface than an `http.ResponseWriter`. Rather than
reducing requests and responses to byte slices, they can be handled as
structured Go types.
