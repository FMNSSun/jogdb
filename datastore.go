package jogdb

import "sync"
import "errors"

type DataStore interface {
	// Returns the value associated with the namespace and document name.
	Get(ns, doc string) ([]byte, error)

	// Sets the value associated with the namespace and document name.
	Put(ns, doc string, v []byte) error

	// Appends to the value associated with the namespace and document
	// name while inserting the specified delimiter in front of the value
	// that is to be appended.
	Append(ns, doc string, delim, v []byte) error

	// Returns true if the token has permission to perform a Get.
	CanGet(token, ns, doc string) (bool, error)

	// Returns true if the token has permission to perform a Put.
	CanPut(token, ns, doc string) (bool, error)

	// Returns true if the token has permission to perform an Append.
	CanAppend(token, ns, doc string) (bool, error)

	// Set permissions for the token for the document and namespace as
	// specified.
	SetToken(token, ns, doc string, get, put, app bool) error

	// Returns true if the token is a namespace admin.
	IsNamespaceAdmin(token, ns string) (bool, error)

	// Returns true if the token is an admin.
	IsAdmin(token string) (bool, error)

	// Adds or removes a token as a namespace admin for the specified namespace.
	// If `is` is true then it adds the token otherwise it removes it. 
	SetNamespaceAdmin(token, ns string, is bool) error

	// Adds or removes a token as an admin.
	// If `is` is true then it adds the token otherwise it removes it. 
	SetAdmin(token string, is bool) error

	// Returns true if the token is root. 
	IsRoot(token string) (bool, error)
}

// This is returned by the Check* functions in case
// there wasn't an 'actual' error but the provided `clientToken`
// simply lacks permission to perform the action. 
var ErrAccessDenied = errors.New("Access denied!")

// Invokes the `SetAdmin` method on `ds` iff `clientToken` is root.
func CheckedSetAdmin(ds DataStore, clientToken, token string, is bool) error {
	ok, err := ds.IsRoot(clientToken)

	if err != nil {
		return err
	}

	if !ok {
		return ErrAccessDenied
	}

	return ds.SetAdmin(token, is)
}

// Invokes the `SetNamespaceAdmin` method on `ds` iff `clientToken` is admin. 
func CheckedSetNamespaceAdmin(ds DataStore, clientToken, token, ns string, is bool) error {
	ok, err := ds.IsAdmin(clientToken)

	if err != nil {
		return err
	}

	if !ok {
		return ErrAccessDenied
	}

	return ds.SetNamespaceAdmin(token, ns, is)
}

// Invokes the `SetToken` method on `ds` iff `clientToken` is namespace admin for the
// specified namespace. 
func CheckedSetToken(ds DataStore, clientToken, token, ns, doc string, get, put, app bool) error {
	ok, err := ds.IsNamespaceAdmin(clientToken, ns)

	if err != nil {
		return err
	}

	if !ok {
		return ErrAccessDenied
	}

	return ds.SetToken(token, ns, doc, get, put, app)
}

// Invokes the `Get` method on `ds` iff `clientToken` has Get permissions.
func CheckedGet(ds DataStore, clientToken, ns, doc string) ([]byte, error) {
	ok, err := ds.CanGet(clientToken, ns, doc)

	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, ErrAccessDenied
	}

	return ds.Get(ns, doc)
}

// Invokes the `Get` method on `ds` iff `clientToken` has Put permissions.
func CheckedPut(ds DataStore, clientToken, ns, doc string, v []byte) error {
	ok, err := ds.CanPut(clientToken, ns, doc)

	if err != nil {
		return err
	}

	if !ok {
		return ErrAccessDenied
	}

	return ds.Put(ns, doc, v)
}

// Invokes the `Get` method on `ds` iff `clientToken` has Append permissions.
func CheckedAppend(ds DataStore, clientToken, ns, doc string, delim, v []byte) error {
	ok, err := ds.CanAppend(clientToken, ns, doc)

	if err != nil {
		return err
	}

	if !ok {
		return ErrAccessDenied
	}

	return ds.Append(ns, doc, delim, v)
}

const permGet = uint8(1)
const permPut = uint8(2)
const permAppend = uint8(4)

type kvBytes map[string][]byte
type kvPerms map[string]uint8
type kvBool map[string]bool
type storageType map[string]kvBytes
type permsType map[string]map[string]kvPerms

type MemDataStore struct {
	storage storageType
	perms permsType
	mutex *sync.Mutex
	nsAdmins map[string]kvBool
	admins kvBool
	rootToken string
}

func NewMemDataStore(rootToken string) *MemDataStore {
	return & MemDataStore {
		storage: make(storageType),
		perms: make(permsType),
		mutex: &sync.Mutex{},
		nsAdmins: make(map[string]kvBool),
		admins: make(kvBool),
		rootToken: rootToken,
	}
}

func (ds *MemDataStore) IsRoot(token string) (bool, error) {
	ds.mutex.Lock()

	is := ds.rootToken == token

	ds.mutex.Unlock()
	return is, nil
}

func (ds *MemDataStore) SetNamespaceAdmin(token, ns string, is bool) error {
	ds.mutex.Lock()

	nsV := ds.nsAdmins[ns]

	if nsV == nil {
		if !is {
			ds.mutex.Unlock()
			return nil // doesn't exist anyway
		} else {
			nsV = make(kvBool)
			ds.nsAdmins[ns] = nsV
		}
	}

	if is {
		nsV[token] = true
	} else {
		delete(nsV, token)
	}

	ds.mutex.Unlock()
	return nil
}

func (ds *MemDataStore) SetAdmin(token string, is bool) error {
	ds.mutex.Lock()

	if is {
		ds.admins[token] = true
	} else {
		delete(ds.admins, token)
	}

	ds.mutex.Unlock()
	return nil
}

func (ds *MemDataStore) IsAdmin(token string) (bool, error) {
	ds.mutex.Lock()

	exists := ds.admins[token]

	if exists {
		ds.mutex.Unlock()
		return true, nil
	}

	ds.mutex.Unlock()
	return false, nil
}

func (ds *MemDataStore) IsNamespaceAdmin(token, ns string) (bool, error) {
	ds.mutex.Lock()

	nsV := ds.nsAdmins[ns]

	if nsV == nil {
		ds.mutex.Unlock()
		return false, nil
	}

	exists := nsV[token]

	if exists {
		ds.mutex.Unlock()
		return true, nil
	}

	ds.mutex.Unlock()
	return false, nil
}

func (ds *MemDataStore) SetToken(token, ns, doc string, get, put, app bool) error {
	ds.mutex.Lock()

	nsV := ds.perms[ns]

	if nsV == nil {
		nsV = make(map[string]kvPerms)
		ds.perms[ns] = nsV
	}

	docV := nsV[doc]

	if docV == nil {
		docV = make(kvPerms)
		nsV[doc] = docV
	}

	if get == false && put == false && app == false {
		delete(docV, token)
	} else {
		curPerms := docV[token]

		if get {
			curPerms |= permGet
		} else {
			curPerms &= ^permGet
		}

		if put {
			curPerms |= permPut
		} else {
			curPerms &= ^permPut
		}

		if app {
			curPerms |= permAppend
		} else {
			curPerms &= ^permAppend
		}

		docV[token] = curPerms
	}

	ds.mutex.Unlock()
	return nil
}

func (ds *MemDataStore) CanGet(token, ns, doc string) (bool, error) {
	ds.mutex.Lock()

	nsV := ds.perms[ns]

	if nsV == nil {
		ds.mutex.Unlock()
		return false, nil
	}

	docV := nsV[doc]

	if docV == nil {
		ds.mutex.Unlock()
		return false, nil
	}

	tokenPerms := docV[token]

	if (tokenPerms & permGet) == permGet {
		ds.mutex.Unlock()
		return true, nil
	} else {
		ds.mutex.Unlock()
		return false, nil
	}
}

func (ds *MemDataStore) CanPut(token, ns, doc string) (bool, error) {
	ds.mutex.Lock()

	nsV := ds.perms[ns]

	if nsV == nil {
		ds.mutex.Unlock()
		return false, nil
	}

	docV := nsV[doc]

	if docV == nil {
		ds.mutex.Unlock()
		return false, nil
	}

	tokenPerms := docV[token]

	if (tokenPerms & permPut) == permPut {
		ds.mutex.Unlock()
		return true, nil
	} else {
		ds.mutex.Unlock()
		return false, nil
	}
}

func (ds *MemDataStore) CanAppend(token, ns, doc string) (bool, error) {
	ds.mutex.Lock()

	nsV := ds.perms[ns]

	if nsV == nil {
		ds.mutex.Unlock()
		return false, nil
	}

	docV := nsV[doc]

	if docV == nil {
		ds.mutex.Unlock()
		return false, nil
	}

	tokenPerms := docV[token]

	if (tokenPerms & permAppend) == permAppend {
		ds.mutex.Unlock()
		return true, nil
	} else {
		ds.mutex.Unlock()
		return false, nil
	}
}

func (ds *MemDataStore) Append(ns, doc string, delim, v []byte) error {
	ds.mutex.Lock()

	nsV := ds.storage[ns]

	if nsV == nil {
		nsV = make(kvBytes)
		ds.storage[ns] = nsV
	}

	nsV[doc] = append(nsV[doc], delim...)
	nsV[doc] = append(nsV[doc], v...)

	ds.mutex.Unlock()

	return nil
}

func (ds *MemDataStore) Put(ns, doc string, v []byte) error {
	ds.mutex.Lock()

	nsV := ds.storage[ns]

	if nsV == nil {
		nsV = make(kvBytes)
		ds.storage[ns] = nsV
	}

	nsV[doc] = v

	ds.mutex.Unlock()

	return nil
}

func (ds *MemDataStore) Get(ns, doc string) ([]byte, error) {
	ds.mutex.Lock()

	nsV := ds.storage[ns]

	if nsV == nil {
		ds.mutex.Unlock()
		return nil, nil
	}

	docV := nsV[doc]

	if docV == nil {
		ds.mutex.Unlock()
		return nil, nil
	}

	ds.mutex.Unlock()
	return docV, nil
}


