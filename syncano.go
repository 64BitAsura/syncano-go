package syncano

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"time"
)

var gLOGGER StdLogger
var gAPIRoot, gAPIServer string

//syncano constants
const (
	APIVersion                = "v1"
	AuthPath                  = "account/auth/"
	AccountPath               = "account/"
	ContentType               = "application/json"
	DefaultTimeOut            = 30
	DefaultAPIRoot            = "https://api.syncano.rocks"
	DefaultServer             = "api.syncano.rocks"
	PostMethod     HTTPMethod = "POST"
	GetMethod      HTTPMethod = "GET"
	PatchMethod    HTTPMethod = "PATCH"
	DeleteMethod   HTTPMethod = "DELETE"
	HeadMethod     HTTPMethod = "HEAD"
)

func init() {
	gAPIRoot = DefaultAPIRoot
	gAPIServer = DefaultServer
}

type httpError struct {
	StatusCode int
}

func (*httpError) RuntimeError() {}

func (e *httpError) Error() string {
	return "Syncano: HTTP Error with status code of " + strconv.Itoa(e.StatusCode)
}

//InfrastructureError type to represent the adapter error - checked excpetion
type InfrastructureError struct {
	error
}

func (e *InfrastructureError) Error() string {
	return e.error.Error()
}

func NewInfrastructureError(msg string) (i *InfrastructureError) {
	i = new(InfrastructureError)
	err := errors.New(msg)
	i.error = err
	return
}

//ClientError type used for the http status code from 400 to 499
type ClientError struct {
	httpError
}

//ServerError type used for the http status code from 500 to 599
type ServerError struct {
	httpError
}

//RedirectionError used for the http status code from 300 to 399
type RedirectionError struct {
	httpError
}

//InformationalError used for the http status code from 100 to 199
type InformationalError struct {
	httpError
}

type authResponse struct {
	APIKey `json:"account_key"`
}

//HTTPMethod type to represent http methods
type HTTPMethod string

//APIKey type to represent the syncano account key
type APIKey string

//InstanceName to represent the syncano instance name
type InstanceName string

//InstanceKey to represent the syncano instance api key
type InstanceKey string

//InstanceDescription type to represent the syncano instance's description
type InstanceDescription string

//Email type to represent the syncano's account id
type Email string

//Password type to represent the syncano account password
type Password string

//Syncano type to represent the syncano
type syncano struct {
	client *http.Client
	apiKey APIKey
	InstanceName
	InstanceKey
	email         Email
	password      Password
	authenticated bool
}

//Syncano type to expose unexported syncanco type's exported methods as type
type Syncano struct {
	syncano
}

//Instance to represent the syncano instance
type Instance struct {
	InstanceName
	InstanceKey
}

//AccountDetails type to represent the syncano account details
type AccountDetails struct {
	ID        int `json:"id"`
	Email     `json:"email"`
	LastName  string `json:"last_name"`
	FirstName string `json:"first_name"`
}

// IsAuthenticated method to check the invoking syncano instance is authenticated!
func (s *syncano) IsAuthenticated() bool {
	return s.authenticated
}

func (s *syncano) validateAPIKEY() (valid bool, err error) {
	accDetails, err := s.GetAccountDetails()
	if err != nil {
		msg := "syncano: Authentication failed for the API KEY - " + string(s.apiKey) + " , more info -" + err.Error()
		gLOGGER.Println(msg)
	}
	valid = accDetails != nil
	return
}

func (s *syncano) GetAccountDetails() (accDetails *AccountDetails, err error) {
	url, _ := url.Parse(gAPIRoot)
	url.Path = APIVersion + "/" + AccountPath
	url.RawQuery = "api_key=" + string(s.apiKey)
	resp, err := s.client.Get(url.String())
	if err != nil {
		return
	}
	var x AccountDetails
	err = parseResponse(resp, &x)
	accDetails = &x
	return
}

func (s *syncano) authenticate() (err error) {
	switch {
	case s.authenticated:
		return
	case s.apiKey != "":
		s.authenticated, err = s.validateAPIKEY()
		return
	case s.email != "" && s.password != "":
		var apiKey APIKey
		apiKey, err = doAuthenticate(s.email, s.password, s.client)
		if err != nil {
			return
		}
		s.apiKey = apiKey
		s.authenticated = true
		return
	default:
		err = NewInfrastructureError("Please sepcify login credentials")
	}
	return
}

func (s *syncano) validateInstance() (result bool, err error) {
	return
}

//ConnectionCredentials type to overried the env specific connection credentials
type ConnectionCredentials struct {
	APIKey
	InstanceKey
	InstanceName
	Email
	Password
	SkipSSLVerification bool
}

type authParam struct {
	Email    `json:"email"`
	Password `json:"password"`
}

func doAuthenticate(email Email, password Password, client *http.Client) (apiKey APIKey, err error) {
	// 1 - pass email and password to Auth path and validate
	// 2 - If it is failed, return an error
	// 3 - If it is passed, return the api key
	url, _ := url.Parse(DefaultAPIRoot)
	url.Path = APIVersion + "/" + AuthPath
	body := &authParam{Email: email, Password: password}
	marshalledBody, _ := json.Marshal(body)
	reader := bytes.NewReader(marshalledBody)
	response, err := client.Post(url.String(), ContentType, reader)
	if err != nil {
		err = NewInfrastructureError("syncano: Request failed - " + err.Error())
		return
	}
	var m authResponse
	if err = parseResponse(response, &m); err != nil {
		return
	}
	apiKey = m.APIKey
	return
}

//GetConnectionCredentialsFromEnv function returns the instance of ConnectionCredentials with properties are from os env
func GetConnectionCredentialsFromEnv() *ConnectionCredentials {
	var email = Email(os.Getenv("SYNCANO_EMAIL"))
	var password = Password(os.Getenv("SYNCANO_PASSWORD"))
	var apiKey = APIKey(os.Getenv("SYNCANO_API_KEY"))
	var skipSSLVerification bool
	if "1" == os.Getenv("SYNCANO_SSL_ENABLED") {
		skipSSLVerification = true
	}
	return &ConnectionCredentials{Email: email, Password: password, APIKey: apiKey, SkipSSLVerification: skipSSLVerification}
}

//Connect function returns the instance of syncano type, if it is authenticated or returns an error
func Connect(connCred *ConnectionCredentials, logger StdLogger) (S *Syncano, err error) {
	gLOGGER = logger
	client := getConn(DefaultServer, connCred.SkipSSLVerification)
	s := syncano{
		client:        client,
		apiKey:        connCred.APIKey,
		InstanceName:  connCred.InstanceName,
		InstanceKey:   connCred.InstanceKey,
		email:         connCred.Email,
		password:      connCred.Password,
		authenticated: false,
	}
	err = s.authenticate()
	if err != nil {
		return nil, err
	}
	S = &Syncano{s}
	return
}

func getConn(serverName string, skipSSLVerify bool) *http.Client {
	/*create an unexported connection func and does following*/
	//1- create tls config based on the ssl verification flag
	//2- create Transport layer and replace tls config to it
	//3- create http client and replace transport layer
	//4- return the client
	tlsConfig := &tls.Config{InsecureSkipVerify: skipSSLVerify, ServerName: serverName}
	transport, _ := http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = tlsConfig
	client := http.DefaultClient
	client.Transport = transport
	client.Timeout = time.Duration(time.Second * DefaultTimeOut)
	return client
}

func parseResponse(response *http.Response, v interface{}) (err error) {
	defer response.Body.Close()
	switch {
	case 400 <= response.StatusCode && response.StatusCode <= 499:
		return &ClientError{httpError: httpError{response.StatusCode}}
	case 500 <= response.StatusCode && response.StatusCode <= 599:
		return &ServerError{httpError: httpError{response.StatusCode}}
	case 300 <= response.StatusCode && response.StatusCode <= 399:
		return &RedirectionError{httpError: httpError{response.StatusCode}}
	case 100 <= response.StatusCode && response.StatusCode <= 199:
		return &InformationalError{httpError: httpError{response.StatusCode}}
	}
	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return NewInfrastructureError("syncano: Error in reading the response body -" + err.Error())
	}
	err = json.Unmarshal(bytes, v)
	if err != nil {
		return NewInfrastructureError("syncano: error in parsing response body bytes - " + string(bytes[:len(bytes)]) + "to type -" + reflect.TypeOf(v).String())
	}
	return
}

// Won't compile if StdLogger can't be realized by a log.Logger
var _ StdLogger = &log.Logger{}

// Won't compile if http.RoundTripper can't be realized by a http.Transport
var _ http.RoundTripper = &http.Transport{}

type StdLogger interface {
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})

	Fatal(...interface{})
	Fatalf(string, ...interface{})
	Fatalln(...interface{})

	Panic(...interface{})
	Panicf(string, ...interface{})
	Panicln(...interface{})
}
