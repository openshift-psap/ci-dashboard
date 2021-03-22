package artifacts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
    "net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	v1 "github.com/openshift-psap/ci-dashboard/api/matrix/v1"
	"github.com/PuerkitoBio/goquery"
)

type ArtifactType string

const (
	TypeJson ArtifactType = "type:json"
	TypeHtml              = "type:html"
	TypeBytes             = "type:bytes"
)

type ArtifactResult struct {
	Json JsonResult
	Html *goquery.Document
	Bytes []byte
}

type JsonResult map[string]interface{}

func fetchRemoveFromCache(test_matrix v1.MatrixSpec, path string) error {
	cache_path := fmt.Sprintf("%s/%s", test_matrix.ArtifactsCache, path)
	return os.Remove(cache_path)
}

func fetchArtifact(test_matrix v1.MatrixSpec, path string) ([]byte, error) {
	cache_path := fmt.Sprintf("%s/%s", test_matrix.ArtifactsCache, path)
	content, err := ioutil.ReadFile(cache_path)
	if err == nil {
		log.Debugf("File %s found in the cache", path)
		return content, nil
	}

	artifact_path := fmt.Sprintf("%s/%s", test_matrix.ArtifactsURL, path)
	log.Debugf("Fetching %s ...", path)
	resp, err := http.Get(artifact_path)
	if err != nil {
		return []byte{}, fmt.Errorf("error fetching %s: %v", artifact_path, err)
	}

	defer resp.Body.Close()
	content, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("error reading %s: %v", artifact_path, err)
	}

	cache_dir, err := filepath.Abs(filepath.Dir(cache_path))
    if err != nil {
		log.Warningf("Failed to get cache directory for %s: %v", cache_path, err)
    }

	err = os.MkdirAll(cache_dir, os.ModePerm)
	if err != nil {
		log.Warningf("Failed to create cache directory %s: %v", cache_dir, err)
		return []byte{}, err
    }

	err = ioutil.WriteFile(cache_path, content, 0644)
	if err != nil {
		log.Warningf("Failed to write into cache file at %s: %v", cache_path, err)
	}

	return content, nil
}

func fetchHtmlArtifact(test_matrix v1.MatrixSpec, path string) (*goquery.Document, error) {
	content, err := fetchArtifact(test_matrix, path)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("error parsing the HTML of %s: %v", path, err)
	}

	return doc, nil
}

func fetchJsonArtifact(test_matrix v1.MatrixSpec, path string) (JsonResult, error){
	content, err := fetchArtifact(test_matrix, path)
	if err != nil {
		return nil, err
	}
	var result JsonResult
	err = json.Unmarshal(content, &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing the JSON of %s: %v", path, err)
	}

	return result, nil
}

func fetchTestResult(test_matrix v1.MatrixSpec, matrix_name, prow_name, build_id, filename string, filetype ArtifactType) (ArtifactResult, error) {
	file_path := fmt.Sprintf("%s/%s/%s", prow_name, build_id, filename)
	var result ArtifactResult
	var err error
	if filetype == TypeJson {
		result.Json, err = fetchJsonArtifact(test_matrix, file_path)
	} else if filetype == TypeJson {
		result.Html, err = fetchHtmlArtifact(test_matrix, file_path)
	} else if filetype == TypeBytes {
		result.Bytes, err = fetchArtifact(test_matrix, file_path)
	} else {
		return result, fmt.Errorf("code error: invalid file type requested %s", filetype)
	}
	if err != nil {
		return result, fmt.Errorf("error fetching the test results from %s: %v", file_path, err)
	}

	return result, nil
}

func FetchLastTestResult(test_matrix v1.MatrixSpec, matrix_name string, test v1.TestSpec, filename string, filetype ArtifactType) (string, ArtifactResult, error) {
	last_test_path := fmt.Sprintf("%s/latest-build.txt", test.ProwName)
	last_test_build_id, err := fetchArtifact(test_matrix, last_test_path)
	if err != nil {
		fetchRemoveFromCache(test_matrix, last_test_path)
		return "", ArtifactResult{}, fmt.Errorf("error fetching the last test build_id from %s: %v",
			last_test_path, err)
	}

	if _, err := strconv.Atoi(string(last_test_build_id)); err != nil {
		fetchRemoveFromCache(test_matrix, last_test_path)
		return "", ArtifactResult{}, fmt.Errorf("error validating the last test build_id from %s: %v",
			last_test_path, err)
	}

	/*
	if err = fetchRemoveFromCache(test_matrix, last_test_path); err != nil {
		log.Warningf("Failed to remove %s from cache : %v", last_test_path, err)
	}
*/
	last_test_file, err := fetchTestResult(test_matrix, matrix_name, test.ProwName,
		string(last_test_build_id), filename, filetype)
	if (err != nil) {
		return "", ArtifactResult{}, fmt.Errorf("error fetching the results of the last test of %s:%s (%s): %v",
			matrix_name, test.ProwName, last_test_build_id, err)

	}

	return string(last_test_build_id), last_test_file, nil
}


func FetchLastNTestResults(test_matrix v1.MatrixSpec, matrix_name, prow_name string, nb_test int, filename string, filetype ArtifactType) ([]string, map[string]ArtifactResult, error) {
	test_list_path := fmt.Sprintf("%s/?index", prow_name)
	test_list_html, err := fetchHtmlArtifact(test_matrix, test_list_path)
	if err != nil {
		return nil, nil, fmt.Errorf("error fetching the tests of %s / %s: %v", matrix_name, prow_name, err)
	}

	test_results := map[string]ArtifactResult{}
	build_ids := make([]string, 0, nb_test)

	test_list_html.Find("li.grid-row").EachWithBreak(func(i int, s *goquery.Selection) bool {
		entry_type, found := s.Find("img").Attr("src")
		if !found || entry_type != "/icons/dir.png" {
			return true
		}
		test_build_id := strings.TrimSuffix(strings.TrimSpace(s.Find("a").Text()), "/")
		if (!found) {
			return true
		}

		build_ids = append([]string{test_build_id}, build_ids...)
		return true
	})
	if len(build_ids) > nb_test {
		build_ids = build_ids[:nb_test]
	}
	for _, test_build_id := range build_ids {
		test_file, err := fetchTestResult(test_matrix, matrix_name, prow_name, test_build_id, filename, filetype)
		if (err != nil) {
			log.Warningf("error fetching the results of %s:%s/%s (%s): %v",
				matrix_name, prow_name, test_build_id, filename, err)

		}

		test_results[test_build_id] = test_file
	}

	return build_ids, test_results, err
}
