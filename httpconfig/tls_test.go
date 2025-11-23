package httpconfig

// NOTE: Run the script at testdata/tls/create-certs.sh before running TLS tests.

// func TestTLS(t *testing.T) {
// 	server := createMockTLSServer(t, false)
// 	defer server.Close()

// 	keyPem, err := os.ReadFile(filepath.Join("testdata/tls/certs", "client.key"))
// 	if err != nil {
// 		t.Fatalf("failed to load client key: %s", err)
// 	}

// 	keyData := base64.StdEncoding.EncodeToString(keyPem)
// 	t.Setenv("TLS_KEY_PEM", string(keyData))

// 	testCases := []struct {
// 		Endpoint   string
// 		ConfigPath string
// 	}{
// 		{
// 			Endpoint:   "/auth/hello",
// 			ConfigPath: "testdata/tls.yaml",
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.ConfigPath, func(t *testing.T) {
// 			config, err := goutils.ReadJSONOrYAMLFile[HTTPClientConfig](tc.ConfigPath)
// 			if err != nil {
// 				t.Fatal(err.Error())
// 			}

// 			client, err := NewClientFromConfig(*config)
// 			if err != nil {
// 				t.Fatal("failed to create client: " + err.Error())
// 			}
// 			defer client.Close()

// 			client.SetBaseURL(server.URL)

// 			resp, err := client.R().Get(tc.Endpoint)
// 			if err != nil {
// 				t.Fatal("failed to get: " + err.Error())
// 			}
// 			defer resp.Body.Close()

// 			if resp.StatusCode() != http.StatusOK {
// 				t.Fatalf("expected HTTP 200, get: %d", resp.StatusCode())
// 			}
// 		})
// 	}
// }

// func TestTLSInsecure(t *testing.T) {
// 	server := createMockTLSServer(t, true)
// 	defer server.Close()

// 	t.Setenv("TLS_INSECURE", "true")

// 	testCases := []struct {
// 		Endpoint   string
// 		ConfigPath string
// 	}{
// 		{
// 			Endpoint:   "/auth/hello",
// 			ConfigPath: "testdata/insecureTLS.yaml",
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.ConfigPath, func(t *testing.T) {
// 			config, err := goutils.ReadJSONOrYAMLFile[RestyConfig](tc.ConfigPath)
// 			if err != nil {
// 				t.Fatal(err.Error())
// 			}

// 			client, err := NewClientFromConfig(*config)
// 			if err != nil {
// 				t.Fatal("failed to create client: " + err.Error())
// 			}
// 			defer client.Close()

// 			client.SetBaseURL(server.URL)

// 			resp, err := client.R().Get(tc.Endpoint)
// 			if err != nil {
// 				t.Fatal("failed to get: " + err.Error())
// 			}
// 			defer resp.Body.Close()

// 			if resp.StatusCode() != http.StatusOK {
// 				t.Fatalf("expected HTTP 200, get: %d", resp.StatusCode())
// 			}
// 		})
// 	}
// }

// func createMockTLSServer(
// 	t *testing.T,
// 	insecure bool,
// ) *httptest.Server {
// 	t.Helper()
// 	mux := http.NewServeMux()

// 	mux.HandleFunc("/auth/hello", func(w http.ResponseWriter, r *http.Request) {
// 		switch r.Method {
// 		case http.MethodGet:
// 			w.WriteHeader(http.StatusOK)
// 		default:
// 			w.WriteHeader(http.StatusMethodNotAllowed)
// 			return
// 		}
// 	})

// 	var tlsConfig *tls.Config

// 	dir := "testdata/tls/certs"

// 	// load CA certificate file and add it to list of client CAs
// 	caCertFile, err := os.ReadFile(filepath.Join(dir, "ca.crt"))
// 	if err != nil {
// 		log.Fatalf("error reading CA certificate: %v", err)
// 	}
// 	caCertPool := x509.NewCertPool()
// 	caCertPool.AppendCertsFromPEM(caCertFile)

// 	// Create the TLS Config with the CA pool and enable Client certificate validation
// 	cert, err := tls.LoadX509KeyPair(
// 		filepath.Join(dir, "server.pem"),
// 		filepath.Join(dir, "server.key"),
// 	)

// 	tlsConfig = &tls.Config{
// 		ClientCAs:          caCertPool,
// 		Certificates:       []tls.Certificate{cert},
// 		ClientAuth:         tls.RequireAndVerifyClientCert,
// 		InsecureSkipVerify: insecure,
// 	}

// 	if insecure {
// 		tlsConfig.ClientAuth = tls.RequestClientCert
// 	}

// 	server := httptest.NewUnstartedServer(mux)
// 	server.TLS = tlsConfig
// 	server.StartTLS()

// 	return server
// }
