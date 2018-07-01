package jogdb

import "github.com/gorilla/mux"
import "net/http"
import "io/ioutil"
import "path/filepath"
import "encoding/json"
import "github.com/FMNSSun/rndstring"

type ApiState struct {
	ContentTypes map[string]string
	DefaultContentType string
	DataStore DataStore
	StringGenerator rndstring.StringGenerator
	Delimiters map[string][]byte
}

func (e *ApiState) generateToken() string {
	return e.StringGenerator.Generate()
}

func getToken(r *http.Request) string {
	return r.Header.Get("X-API-TOKEN")
}

func checkErrJSON(err error, w http.ResponseWriter) bool {
	if err != nil {
		http.Error(w, "ErrJSON: Your request contained invalid JSON.", http.StatusBadRequest)
		return false
	}

	return true
}

func returnJSON(v interface{}, w http.ResponseWriter) {
	b, err := json.Marshal(v)

	if err != nil {
		http.Error(w, "ErrJSON: These was an internal error. Contact administrator or try again.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func checkErr(err error, w http.ResponseWriter) bool {
	if err == ErrAccessDenied {
		http.Error(w, "AccessDenied: Either no X-API-TOKEN was supplied or you don't have permissions for this action.", http.StatusForbidden)
		return false
	}

	if err != nil {
		http.Error(w, "ErrPut: There was an internal error. Contact administrator or try again.", http.StatusInternalServerError)
		return false
	}

	return true
}

func readRequest(w http.ResponseWriter, r *http.Request) []byte {
	b, err := ioutil.ReadAll(r.Body)

	if err != nil {
		http.Error(w, "ErrReadingRequest: There was an error reading your request.", http.StatusInternalServerError)
		return nil
	}

	return b
}

func (e *ApiState) index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("jogdb api"))
}

func (e *ApiState) putDoc(w http.ResponseWriter, r *http.Request) {
	b := readRequest(w, r)

	if b == nil {
		return
	}

	clientToken := getToken(r)
	vars := mux.Vars(r)
	ns, doc := vars["ns"], vars["doc"]

	err := CheckedPut(e.DataStore, clientToken, ns, doc, b)

	if !checkErr(err, w) {
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("OK"))
}

func (e *ApiState) appendDoc(w http.ResponseWriter, r *http.Request) {
	b := readRequest(w, r)

	if b == nil {
		return
	}

	clientToken := getToken(r)
	vars := mux.Vars(r)
	ns, doc := vars["ns"], vars["doc"]

	ext := filepath.Ext(doc)

	delim := e.Delimiters[ext]

	if delim == nil {
		delim = []byte{}
	}

	err := CheckedAppend(e.DataStore, clientToken, ns, doc, delim, b)

	if !checkErr(err, w) {
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("OK"))
}

func (e *ApiState) getDoc(w http.ResponseWriter, r *http.Request) {
	clientToken := getToken(r)
	vars := mux.Vars(r)
	ns, doc := vars["ns"], vars["doc"]

	v, err := CheckedGet(e.DataStore, clientToken, ns, doc)

	if !checkErr(err, w) {
		return
	}

	if v == nil {
		http.Error(w, "ErrNotFound: The resource you requested could not be found.", http.StatusNotFound)
		return
	}

	ct := e.ContentTypes[filepath.Ext(doc)]

	if ct == "" {
		ct = e.DefaultContentType
	}

	w.Header().Set("Content-Type", ct)
	w.Write(v)
}

type setTokenRequest struct {
	Token string
	Put bool
	Get bool
	Append bool
}

func (e *ApiState) setToken(w http.ResponseWriter, r *http.Request) {
	clientToken := getToken(r)
	vars := mux.Vars(r)
	ns, doc := vars["ns"], vars["doc"]

	b := readRequest(w, r)

	if b == nil {
		return
	}

	var str setTokenRequest
	err := json.Unmarshal(b, &str)

	if !checkErrJSON(err, w) {
		return
	}

	if str.Token == "" {
		str.Token = e.generateToken()
	}

	err = CheckedSetToken(e.DataStore, clientToken, str.Token, ns, doc, str.Put, str.Get, str.Append)

	if !checkErr(err, w) {
		return
	}

	returnJSON(str, w)
}

type setNamespaceAdminRequest struct {
	Token string
	Is bool
}

func (e *ApiState) setNamespaceAdmin(w http.ResponseWriter, r *http.Request) {
	clientToken := getToken(r)
	vars := mux.Vars(r)
	ns := vars["ns"]

	b := readRequest(w, r)

	if b == nil {
		return
	}

	var snar setNamespaceAdminRequest
	err := json.Unmarshal(b, &snar)

	if !checkErrJSON(err, w) {
		return
	}

	if snar.Token == "" {
		snar.Token = e.generateToken()
	}

	err = CheckedSetNamespaceAdmin(e.DataStore, clientToken, snar.Token, ns, snar.Is)

	if !checkErr(err, w) {
		return
	}

	returnJSON(snar, w)
}

type setAdminRequest struct {
	Token string
	Is bool
}

func (e *ApiState) setAdmin(w http.ResponseWriter, r *http.Request) {
	clientToken := getToken(r)

	b := readRequest(w, r)

	if b == nil {
		return
	}

	var sar setAdminRequest
	err := json.Unmarshal(b, &sar)

	if !checkErrJSON(err, w) {
		return
	}

	if sar.Token == "" {
		sar.Token = e.generateToken()
	}

	err = CheckedSetAdmin(e.DataStore, clientToken, sar.Token, sar.Is)

	if !checkErr(err, w) {
		return
	}

	returnJSON(sar, w)
}


func NewAPI(e *ApiState) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/", e.index).Methods("GET")
	r.HandleFunc("/r/{ns}/{doc}", e.appendDoc).Methods("PUT")
	r.HandleFunc("/r/{ns}/{doc}", e.putDoc).Methods("POST")
	r.HandleFunc("/r/{ns}/{doc}", e.getDoc).Methods("GET")
	r.HandleFunc("/m/token/{ns}/{doc}", e.setToken).Methods("PUT")
	r.HandleFunc("/m/admin/{ns}", e.setNamespaceAdmin).Methods("PUT")
	r.HandleFunc("/m/admin", e.setAdmin).Methods("PUT")

	return r
}
