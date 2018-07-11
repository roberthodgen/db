# Golang db.Session
> Wrapper for mgo.Session with auto connection cloning and closing.

**Work in progress.**


## Idea

Write queries (in the form of `db.Query`: `func (*mgo.Collection) error`) and pass them into a Session's Do method.

Your underling mgo.Session will be automatically cloned (or dialed), Query executed, and session closed.

```go
// session var from db.NewSession(*mgo.DialInfo)
// Configure database and collection with session.WithDB("mydb") or session.WithCollection("users")

query := func(c *mgo.Collection) error {
	return c.Insert(user)
}

// The underlying mgo.Session will be automatically dialed, cloned, and closed.
err := session.Do(query)
```


## Example

You're writing some CRUD operations for a user and define an interface like:

```go
type User interface {
	Create(*User) error
	Read(string) (*User, error)
	Update(*User) error
	Delete(string) error
}
```

A simple implementation using this db.Session's `Do(Query) error` looks like:

```go
type store struct {
	session *db.Session
}

func (s *store) Create(u *User) error {
	return s.session.Do(func(c *mgo.Collection) error {
		return c.Insert(u)
	})
}

func (s *store) Read(id string) (*User, error) {
	var u *User
	query := func(c *mgo.Collection) error {
		return c.Find(bson.M{"id": id}).One(&u)
	}

	return u, s.session.Do(query)
}

func (s *store) Update(u *User) error {
	return s.session.Do(func(c *mgo.Collection) error {
		return c.Update(bson.M{"id": u.Id}, u)
	})
}

func (s *store) Delete(id string) error {
	return s.session.Do(func(c *mgo.Collection) error {
		return c.Remove(bson.M{"id": id})
	})
}
```

Note the session is cloned automatically in `Do` and closed there to.
Any connection errors will be return along with the query (actually, a connection error means the Query function is never called).

When `Do(Query) error` is called:
1. Check if there's a mgo.Session live, if so clone it, otherwise create it. Any dialing errors would be returned.
2. Run the Query (`func (c *mgo.Collection) error`) and return its result.
3. Close the cloned connection.
