package syncano

import (
	"bytes"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"log"
	"net/http"
	"testing"
	"time"
)

func init() {
	gAPIRoot = "https://api.syncano.rocks"
	gAPIServer = "api.syncano.rocks"
}

// account from the api.syncano.rocks
var email Email = "samkumar15@gmail.com"
var password Password = "syncanoTest"
var apiKey APIKey = "5dbf5c7f3f5aca5213d87ab06db08092a6780706"

func TestSyncanoAPI(t *testing.T) {
	var buf bytes.Buffer
	defer fmt.Print(&buf)

	Convey("Validate getConnection func", t, func() {
		var client *http.Client
		Convey("Create a connection with validate serverName with ssl verification", func() {
			client = getConn(DefaultServer, true)
			config := client.Transport.(*http.Transport).TLSClientConfig
			So(config.ServerName, ShouldEqual, DefaultServer)
			So(true, ShouldEqual, config.InsecureSkipVerify)
			So(client.Timeout, ShouldResemble, time.Duration(time.Second*DefaultTimeOut))
			Convey("Validate with simple get request", func() {
				response, err := client.Get(gAPIRoot + "/")
				So(err, ShouldBeNil)
				So(response, ShouldNotBeNil)
			})
		})
	})

	Convey("Create a syncano instance", t, func() {
		logger := log.New(&buf, "logger: ", log.Lshortfile)
		Convey("Using valid email and password", func() {
			syncano, err := Connect(&ConnectionCredentials{Email: email, Password: password, SkipSSLVerification: true}, logger)
			So(err, ShouldBeNil)
			Convey("syncano instance should be authenticated", func() {
				So(syncano.IsAuthenticated(), ShouldEqual, true)
			})
			Convey("API Key shouldn't be empty", func() {
				So(syncano.apiKey, ShouldNotEqual, "")
			})
		})

		Convey("Using API Key", func() {
			syncano, err := Connect(&ConnectionCredentials{SkipSSLVerification: true, APIKey: apiKey}, logger)
			So(err, ShouldBeNil)
			Convey("syncano instance should be authenticated", func() {
				So(syncano.IsAuthenticated(), ShouldEqual, true)
			})
			Convey("API Key should be equal to the API Key to authenticate", func() {
				So(syncano.apiKey, ShouldEqual, apiKey)
			})
		})

		Convey("Using invalid email and password", func() {
			syncano, err := Connect(&ConnectionCredentials{SkipSSLVerification: true, Email: "big.boy@email.com", Password: "password"}, logger)
			So(err, ShouldNotBeNil)
			Convey("syncano instance be nil", func() {
				So(syncano, ShouldBeNil)
			})
		})

		Convey("Using invalid API key", func() {
			syncano, err := Connect(&ConnectionCredentials{SkipSSLVerification: true, APIKey: "xx-xxx-xxxx-xxxxx"}, logger)
			So(err, ShouldNotEqual, nil)
			Convey("syncano instance be nil", func() {
				So(syncano, ShouldEqual, nil)
			})
		})
	})
}
