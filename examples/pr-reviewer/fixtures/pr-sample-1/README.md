# Fixture: pr-sample-1

A small Scala 3 PR with deliberate smells across all three review dimensions,
so each specialist has something concrete to find.

## Planted smells

### Types (for `reviewer-types`)
- `lookup(id: String)` — stringly-typed user ID; should be an opaque
  `UserId` type.
- `lookup(...): Future[Any]` — returning `Any` erases the result shape; should
  be `Future[Option[User]]` or a sealed ADT.
- `lookup` returns `null` for the not-found case instead of `Option`.
- `expireStale(cutoffEpochMillis: String)` — should be a `java.time.Instant`
  or at least a `Long`; the stringly-typed epoch forces a parse at the
  boundary.

### Idioms (for `reviewer-idioms`)
- `implicit ec: ExecutionContext` — Scala 3 idiom is `using ec: ExecutionContext`.
- `scala.collection.JavaConverters._` — deprecated; use
  `scala.jdk.CollectionConverters`.
- Hand-assembled `Map[String, String]` where a `case class` or an `enum User`
  plus `derives` would be idiomatic.

### Semantics (for `reviewer-semantics`)
- `rolesFor` uses string concatenation to build a SQL query — classic SQL
  injection (Critical).
- `expireStale` drops the `Future`'s failure case: `.foreach` on a failed
  future never fires, so errors silently vanish.
- `lookup` never closes the `PreparedStatement` or `ResultSet` — resource leak.
- `rolesFor`'s iteration calls `javaList.next()` on the `ResultSet` but the
  variable is named as if it were a Java `List`, hiding the statement leak.

The synthesizer should produce a `summary.md` that groups these by severity
across dimensions (expect at least one Critical — the SQL injection).
