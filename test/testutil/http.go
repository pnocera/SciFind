package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// HTTPTestUtil provides HTTP testing utilities
type HTTPTestUtil struct {
	router *gin.Engine
	server *httptest.Server
}

// SetupTestHTTPServer creates a test HTTP server with Gin router
func SetupTestHTTPServer(t *testing.T) *HTTPTestUtil {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Add basic middleware for testing
	router.Use(gin.Recovery())
	
	return &HTTPTestUtil{
		router: router,
	}
}

// Router returns the Gin router
func (h *HTTPTestUtil) Router() *gin.Engine {
	return h.router
}

// StartServer starts the test server
func (h *HTTPTestUtil) StartServer() {
	h.server = httptest.NewServer(h.router)
}

// StopServer stops the test server
func (h *HTTPTestUtil) StopServer() {
	if h.server != nil {
		h.server.Close()
	}
}

// GetServerURL returns the test server URL
func (h *HTTPTestUtil) GetServerURL() string {
	if h.server != nil {
		return h.server.URL
	}
	return ""
}

// MakeRequest makes an HTTP request to the test server
func (h *HTTPTestUtil) MakeRequest(t *testing.T, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody io.Reader
	
	if body != nil {
		switch v := body.(type) {
		case string:
			reqBody = bytes.NewBufferString(v)
		case []byte:
			reqBody = bytes.NewBuffer(v)
		default:
			jsonBody, err := json.Marshal(body)
			require.NoError(t, err)
			reqBody = bytes.NewBuffer(jsonBody)
		}
	}

	req, err := http.NewRequest(method, path, reqBody)
	require.NoError(t, err)

	// Set default content type for JSON
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	h.router.ServeHTTP(recorder, req)

	return recorder
}

// MakeJSONRequest makes a JSON request and returns the response
func (h *HTTPTestUtil) MakeJSONRequest(t *testing.T, method, path string, body interface{}) *httptest.ResponseRecorder {
	return h.MakeRequest(t, method, path, body, map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	})
}

// AssertJSONResponse asserts the response is JSON and unmarshals it
func (h *HTTPTestUtil) AssertJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, target interface{}) {
	require.Equal(t, expectedStatus, recorder.Code)
	require.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))

	if target != nil {
		err := json.Unmarshal(recorder.Body.Bytes(), target)
		require.NoError(t, err)
	}
}

// AssertErrorResponse asserts the response is an error with specific message
func (h *HTTPTestUtil) AssertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	require.Equal(t, expectedStatus, recorder.Code)
	
	var errorResp map[string]interface{}
	err := json.Unmarshal(recorder.Body.Bytes(), &errorResp)
	require.NoError(t, err)
	
	if expectedMessage != "" {
		require.Contains(t, errorResp, "error")
		require.Equal(t, expectedMessage, errorResp["error"])
	}
}

// CreateMockHTTPServer creates a mock HTTP server for external API testing
func CreateMockHTTPServer(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	mux := http.NewServeMux()
	
	for path, handler := range handlers {
		mux.HandleFunc(path, handler)
	}
	
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	
	return server
}

// CreateMockArxivServer creates a mock ArXiv API server
func CreateMockArxivServer(t *testing.T) *httptest.Server {
	return CreateMockHTTPServer(t, map[string]http.HandlerFunc{
		"/api/query": func(w http.ResponseWriter, r *http.Request) {
			// Mock ArXiv API response
			response := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>ArXiv Query: search_query=all:machine+learning</title>
  <id>http://arxiv.org/api/query</id>
  <updated>2024-01-01T00:00:00-05:00</updated>
  <opensearch:totalResults xmlns:opensearch="http://a9.com/-/spec/opensearch/1.1/">1</opensearch:totalResults>
  <opensearch:startIndex xmlns:opensearch="http://a9.com/-/spec/opensearch/1.1/">0</opensearch:startIndex>
  <opensearch:itemsPerPage xmlns:opensearch="http://a9.com/-/spec/opensearch/1.1/">1</opensearch:itemsPerPage>
  <entry>
    <id>http://arxiv.org/abs/2301.00001v1</id>
    <updated>2023-01-01T00:00:00-05:00</updated>
    <published>2023-01-01T00:00:00-05:00</published>
    <title>Test Paper: Machine Learning Advances</title>
    <summary>This is a test paper about machine learning advances.</summary>
    <author>
      <name>Test Author</name>
    </author>
    <arxiv:doi xmlns:arxiv="http://arxiv.org/schemas/atom">10.1000/test.001</arxiv:doi>
    <link title="doi" href="http://dx.doi.org/10.1000/test.001" rel="related"/>
    <arxiv:primary_category xmlns:arxiv="http://arxiv.org/schemas/atom" term="cs.AI" scheme="http://arxiv.org/schemas/atom"/>
    <category term="cs.AI" scheme="http://arxiv.org/schemas/atom"/>
    <category term="cs.ML" scheme="http://arxiv.org/schemas/atom"/>
  </entry>
</feed>`
			w.Header().Set("Content-Type", "application/atom+xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
		},
	})
}

// CreateMockSemanticScholarServer creates a mock Semantic Scholar API server
func CreateMockSemanticScholarServer(t *testing.T) *httptest.Server {
	return CreateMockHTTPServer(t, map[string]http.HandlerFunc{
		"/graph/v1/paper/search": func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"total": 1,
				"offset": 0,
				"next": nil,
				"data": []map[string]interface{}{
					{
						"paperId": "test-paper-id",
						"title": "Test Paper: Semantic Scholar",
						"abstract": "This is a test paper from Semantic Scholar API.",
						"authors": []map[string]interface{}{
							{
								"authorId": "test-author-id",
								"name": "Test Author",
							},
						},
						"year": 2023,
						"venue": "Test Journal",
						"citationCount": 42,
						"referenceCount": 25,
						"fieldsOfStudy": []string{"Computer Science", "Machine Learning"},
						"url": "https://semanticscholar.org/paper/test-paper-id",
					},
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		},
	})
}

// WithTestContext adds a test context to the Gin router
func (h *HTTPTestUtil) WithTestContext(t *testing.T, fn func(*gin.Context)) {
	h.router.GET("/test", func(c *gin.Context) {
		fn(c)
	})
}

// RequestBuilder helps build HTTP requests for testing
type RequestBuilder struct {
	method  string
	path    string
	body    interface{}
	headers map[string]string
	query   map[string]string
}

// NewRequestBuilder creates a new request builder
func NewRequestBuilder(method, path string) *RequestBuilder {
	return &RequestBuilder{
		method:  method,
		path:    path,
		headers: make(map[string]string),
		query:   make(map[string]string),
	}
}

// WithBody sets the request body
func (rb *RequestBuilder) WithBody(body interface{}) *RequestBuilder {
	rb.body = body
	return rb
}

// WithHeader adds a header
func (rb *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	rb.headers[key] = value
	return rb
}

// WithQuery adds a query parameter
func (rb *RequestBuilder) WithQuery(key, value string) *RequestBuilder {
	rb.query[key] = value
	return rb
}

// WithJSONBody sets JSON body and content type
func (rb *RequestBuilder) WithJSONBody(body interface{}) *RequestBuilder {
	rb.body = body
	rb.headers["Content-Type"] = "application/json"
	return rb
}

// WithAuth adds authorization header
func (rb *RequestBuilder) WithAuth(token string) *RequestBuilder {
	rb.headers["Authorization"] = "Bearer " + token
	return rb
}

// Execute executes the request using the HTTP test utility
func (rb *RequestBuilder) Execute(t *testing.T, httpUtil *HTTPTestUtil) *httptest.ResponseRecorder {
	path := rb.path
	if len(rb.query) > 0 {
		path += "?"
		first := true
		for key, value := range rb.query {
			if !first {
				path += "&"
			}
			path += key + "=" + value
			first = false
		}
	}
	
	return httpUtil.MakeRequest(t, rb.method, path, rb.body, rb.headers)
}