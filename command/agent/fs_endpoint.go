package agent

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

var (
	allocIDNotPresentErr  = fmt.Errorf("must provide a valid alloc id")
	fileNameNotPresentErr = fmt.Errorf("must provide a file name")
)

func (s *HTTPServer) DirectoryListRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	var allocID, path string

	if allocID = strings.TrimPrefix(req.URL.Path, "/v1/client/fs/ls/"); allocID == "" {
		return nil, allocIDNotPresentErr
	}
	if path = req.URL.Query().Get("path"); path == "" {
		path = "/"
	}
	return s.agent.client.FSList(allocID, path)
}

func (s *HTTPServer) FileStatRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	var allocID, path string
	if allocID = strings.TrimPrefix(req.URL.Path, "/v1/client/fs/stat/"); allocID == "" {
		return nil, allocIDNotPresentErr
	}
	if path := req.URL.Query().Get("path"); path == "" {
		return nil, fileNameNotPresentErr
	}
	return s.agent.client.FSStat(allocID, path)
}

func (s *HTTPServer) FileReadAtRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	var allocID, path string
	var offset, limit int64
	var err error

	q := req.URL.Query()

	if allocID = strings.TrimPrefix(req.URL.Path, "/v1/client/fs/readat/"); allocID == "" {
		return nil, allocIDNotPresentErr
	}
	if path = q.Get("path"); path == "" {
		return nil, fileNameNotPresentErr
	}

	if offset, err = strconv.ParseInt(q.Get("offset"), 10, 64); err != nil {
		return nil, fmt.Errorf("error parsing offset: %v", err)
	}
	if limit, err = strconv.ParseInt(q.Get("limit"), 10, 64); err != nil {
		return nil, fmt.Errorf("error parsing limit: %v", err)
	}
	if err = s.agent.client.FSReadAt(allocID, path, offset, limit, resp); err != nil {
		return nil, err
	}
	return nil, nil
}
