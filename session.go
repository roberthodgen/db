// Package db provides an abstraction around mgo and MongoDB sessions.
//
// Auto session cloning and closing around when a Query is run by Do.
//
// It is heavily based off this post: http://denis.papathanasiou.org/archive/2012.10.14.post.pdf
//
// Simple goal of making interacting with MongoDB and the mgo package trivial.
// A Session wraps a mgo.Session with a mutex lock for initial creation.
// The mgo.Session is lazily created and dialed.
package db

import (
	"gopkg.in/mgo.v2"
	"sync"
)

// Session provides an abstraction to MongoDB.
// Create a new Session via NewSession
type Session struct {
	mgoSession  *mgo.Session
	dialInfo    *mgo.DialInfo
	sessionLock sync.Mutex
	database    *mgo.Database
	collection  *mgo.Collection
}

// NewSession returns a new Session type. Use this method to create Sessions.
// Required information may include database and dial addresses:
//
//  &mgo.DialInfo{
//  	Addrs:    []string{"localhost:27017"},
//  	Timeout:  500 * time.Millisecond,
//  	FailFast: true,
//  }
//
// If Database is passed into info it will be used by the created Session.
func NewSession(info *mgo.DialInfo) *Session {
	sess := &Session{dialInfo: info}
	if info.Database != "" {
		sess = sess.WithDB(info.Database)
	}

	return sess
}

// WithDB returns a new Session with a database set.
// All subsequent queries will be run against that database.
//
// The database may be configured later to allow sharing of a
// single mgo.Session (cloning).
//
// NOTE: After WithDB the collection should be set!
func (s *Session) WithDB(name string) *Session {
	scopy := *s
	scopy.database = &mgo.Database{Name: name}
	return &scopy
}

// WithCollection returns a new Session with a collection set.
// All subsequent queries will be run against that collection.
//
// The collection may be configured later to allow sharing of a
// single mgo.Session (cloning).
//
// NOTE: If the Database changes this should be reconfigured too!
func (s *Session) WithCollection(name string) *Session {
	scopy := *s
	scopy.collection = scopy.database.C(name)
	return &scopy
}

func (s *Session) getSession() (*mgo.Session, error) {
	if s.mgoSession == nil {
		var err error
		s.sessionLock.Lock()
		s.mgoSession, err = mgo.DialWithInfo(s.dialInfo)
		s.sessionLock.Unlock()
		if err != nil {
			return nil, err
		}
	}

	return s.mgoSession.Clone(), nil
}

// Query defines an interface for the query functions.
// Takes in a *mgo.collection from a new session and returns an error.
type Query func(*mgo.Collection) error

// Do runs the Query function with a given collection. Automatically creating
// a cloned session that's closed after the execution of the Query function.
//
// Prior to calling Do it's important to configure a database and collection.
// Otherwise your app will crash.
//
// Example usage:
//
//  sess.Do(func (c *mgo.collection) error {
//  	c.Find(&User{Id: id}).One(&u)
//  })
func (s *Session) Do(q Query) error {
	session, err := s.getSession()
	if err != nil {
		return err
	}

	defer session.Close()
	return q(s.collection.With(session))
}

// Ping runs a trivial ping command just to get in touch with the server.
// Wrapper for mgo.Session's built-in Ping. Automatically creates a cloned
// session that's closed after the Ping operation.
func (s *Session) Ping() error {
	session, err := s.getSession()
	if err != nil {
		return err
	}

	defer session.Close()
	return session.Ping()
}

// Close closes the underlying mgo.Session session, if present.
func (s *Session) Close() {
	if s.mgoSession != nil {
		s.mgoSession.Close()
	}
}
